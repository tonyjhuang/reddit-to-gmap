package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/tonyjhuang/reddit-to-gmap/cache"
	"github.com/tonyjhuang/reddit-to-gmap/csv"
	"github.com/tonyjhuang/reddit-to-gmap/gemini"
	"github.com/tonyjhuang/reddit-to-gmap/maps"
	"github.com/tonyjhuang/reddit-to-gmap/reddit"
)

var (
	subreddit     string
	numPosts      int
	useCache      bool
	timeRange     string
	mapsQueryHint string
	numOutput     int
)

type Config struct {
	RedditClientID     string `env:"REDDIT_CLIENT_ID,required"`
	RedditClientSecret string `env:"REDDIT_CLIENT_SECRET,required"`
	GoogleMapsAPIKey   string `env:"GOOGLE_MAPS_API_KEY,required"`
	GoogleGeminiAPIKey string `env:"GOOGLE_GEMINI_API_KEY,required"`
}

var cfg Config

var rootCmd = &cobra.Command{
	Use:   "reddit-to-gmap",
	Short: "A CLI tool to export Reddit posts and generate Google Maps links",
	Long:  `A CLI tool that allows you to export Reddit posts and generate Google Maps links from location data.`,
}

var exportRedditCmd = &cobra.Command{
	Use:   "debug:export-reddit",
	Short: "Debug: Export top posts from a subreddit to a local cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exportReddit(subreddit, numPosts, useCache)
		return err
	},
}

var exportRestaurantDataCmd = &cobra.Command{
	Use:   "debug:export-restaurant-data",
	Short: "Debug: Parse Reddit posts into structured restaurant data",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exportRestaurantData(subreddit, numPosts, useCache)
		return err
	},
}

var exportFullRestaurantDataCmd = &cobra.Command{
	Use:   "debug:export-full-restaurant-data",
	Short: "Debug: Pull canonical restaurant data from Google Maps API",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exportFullRestaurantData(subreddit, numPosts, useCache)
		return err
	},
}

var generateTopPostGoogleMapCSVCmd = &cobra.Command{
	Use:   "generate-top-post-google-map-csv",
	Short: "Generate a CSV file from top Reddit posts for importing into a custom Google Map",
	RunE: func(cmd *cobra.Command, args []string) error {
		return exportToCSV(subreddit, numPosts, useCache)
	},
}

func init() {
	rootCmd.AddCommand(exportRedditCmd)
	rootCmd.AddCommand(exportRestaurantDataCmd)
	rootCmd.AddCommand(exportFullRestaurantDataCmd)
	rootCmd.AddCommand(generateTopPostGoogleMapCSVCmd)

	// Add flags to all commands
	for _, cmd := range []*cobra.Command{exportRedditCmd, exportRestaurantDataCmd, exportFullRestaurantDataCmd, generateTopPostGoogleMapCSVCmd} {
		cmd.Flags().StringVarP(&subreddit, "subreddit", "s", "", "Subreddit to fetch posts from (required)")
		cmd.Flags().IntVarP(&numPosts, "num-posts", "n", 10, "Number of posts to fetch")
		cmd.Flags().StringVarP(&timeRange, "time-range", "t", "month", "Time range for posts (hour, day, week, month, year, all)")
		cmd.Flags().StringVarP(&mapsQueryHint, "maps-query-hint", "l", "", "Location hint for Google Maps queries (e.g. 'NYC', 'San Francisco')")
		cmd.MarkFlagRequired("subreddit")
	}

	// Add use-cache flag to export commands
	for _, cmd := range []*cobra.Command{exportRedditCmd, exportRestaurantDataCmd, exportFullRestaurantDataCmd, generateTopPostGoogleMapCSVCmd} {
		cmd.Flags().BoolVar(&useCache, "use-cache", true, "Whether to use cached data if available")
	}

	// Add num-output flag to CSV generation command
	generateTopPostGoogleMapCSVCmd.Flags().IntVarP(&numOutput, "num-output", "o", 0, "Maximum number of rows to write to the CSV (0 means no limit)")
}

func main() {
	var err error
	if err = godotenv.Load(); err != nil {
		fmt.Printf("Warning: Could not load .env file: %v (this is OK if environment variables are set directly)\n", err)
	}
	cfg, err = env.ParseAs[Config]()
	if err != nil {
		fmt.Printf("Error parsing environment variables: %+v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getCachedOrFetch is a generic helper function that handles caching logic for any type T
func getCachedOrFetch[T any](cacheKey string, useCache bool, fetchFn func() (T, error)) (T, error) {
	var result T

	// Check cache first if enabled
	if useCache && cache.CacheExists(cacheKey) {
		cacheData, err := cache.ReadFromCache(cacheKey)
		if err != nil {
			return result, fmt.Errorf("error reading from cache: %v", err)
		}

		// Convert cached data back to type T using JSON marshaling/unmarshaling
		jsonData, err := json.Marshal(cacheData.Data)
		if err != nil {
			return result, fmt.Errorf("error marshaling cache data: %v", err)
		}

		if err := json.Unmarshal(jsonData, &result); err != nil {
			return result, fmt.Errorf("error unmarshaling cache data: %v", err)
		}

		fmt.Printf("Found %d items in cache for %s\n", reflect.ValueOf(result).Len(), cacheKey)
		return result, nil
	}

	// Fetch fresh data
	result, err := fetchFn()
	if err != nil {
		return result, err
	}

	// Cache the result
	if err := cache.WriteToCache(cacheKey, result); err != nil {
		return result, fmt.Errorf("error writing to cache: %v", err)
	}

	return result, nil
}

// exportReddit fetches Reddit posts and caches them. Returns the fetched posts.
func exportReddit(subreddit string, numPosts int, useCache bool) ([]reddit.Post, error) {
	return getCachedOrFetch(
		subreddit,
		useCache,
		func() ([]reddit.Post, error) {
			client := reddit.NewClient(cfg.RedditClientID, cfg.RedditClientSecret)
			posts, err := client.GetPosts(subreddit, numPosts, timeRange)
			if err != nil {
				return nil, fmt.Errorf("error fetching posts: %v", err)
			}
			fmt.Printf("Successfully exported %d posts from r/%s (time range: %s)\n", len(posts), subreddit, timeRange)
			return posts, nil
		},
	)
}

// exportRestaurantData processes Reddit posts into restaurant data and caches the results.
// Returns the processed restaurant data.
func exportRestaurantData(subreddit string, numPosts int, useCache bool) ([]gemini.Restaurant, error) {
	restaurantCacheKey := subreddit + "_restaurants"
	return getCachedOrFetch(
		restaurantCacheKey,
		useCache,
		func() ([]gemini.Restaurant, error) {
			fmt.Printf("parsing reddit data with gemini...\n")
			// Get Reddit posts using exportReddit
			posts, err := exportReddit(subreddit, numPosts, useCache)
			if err != nil {
				return nil, err
			}

			// Create a Gemini client
			ctx := context.Background()
			geminiClient, err := gemini.NewClient(ctx, cfg.GoogleGeminiAPIKey)
			if err != nil {
				return nil, fmt.Errorf("error creating Gemini client: %v", err)
			}
			defer geminiClient.Close()

			// Process posts in chunks of 100
			const chunkSize = 100
			var allRestaurants []gemini.Restaurant

			for i := 0; i < len(posts); i += chunkSize {
				end := i + chunkSize
				if end > len(posts) {
					end = len(posts)
				}
				chunk := posts[i:end]

				// Process the chunk with Gemini
				restaurantData, err := geminiClient.ToRestaurantData(ctx, chunk)
				if err != nil {
					return nil, fmt.Errorf("error processing posts chunk with Gemini: %v", err)
				}

				allRestaurants = append(allRestaurants, restaurantData...)
				fmt.Printf("Processed chunk %d/%d posts\n", end, len(posts))
			}

			// Sort all restaurants by upvotes in descending order
			sort.Slice(allRestaurants, func(i, j int) bool {
				return allRestaurants[i].Upvotes > allRestaurants[j].Upvotes
			})

			var uniqueRestaurants = dedupeRestaurants(allRestaurants)

			fmt.Printf("Successfully exported %d restaurants from r/%s\n", len(uniqueRestaurants), subreddit)
			return uniqueRestaurants, nil
		},
	)
}

// dedupeRestaurants removes duplicate Restaurant entries based on the Name field.
// It preserves the order of the first occurrence of each unique restaurant.
// It returns a new slice containing only the unique restaurants.
func dedupeRestaurants(restaurants []gemini.Restaurant) []gemini.Restaurant {
	seen := make(map[string]struct{})

	// Initialize a new slice to store the unique restaurants.
	uniqueRestaurants := make([]gemini.Restaurant, 0, len(restaurants)/2) // Example capacity

	for _, r := range restaurants {
		// Check if we've already seen a restaurant with this name
		if _, found := seen[r.Name]; !found {
			// If this name hasn't been seen before:
			// 1. Mark it as seen by adding it to the map.
			seen[r.Name] = struct{}{}
			// 2. Append the current restaurant to our result slice.
			uniqueRestaurants = append(uniqueRestaurants, r)
		}
		// If the name was already 'found' in the 'seen' map, we simply skip
		// this element, effectively dropping the duplicate.
	}

	return uniqueRestaurants
}

// exportFullRestaurantData processes Reddit posts into restaurant data with canonicalized Google Maps links.
// Returns the processed restaurant data.
func exportFullRestaurantData(subreddit string, numPosts int, useCache bool) ([]maps.Restaurant, error) {
	fullRestaurantCacheKey := subreddit + "_full_restaurants"
	return getCachedOrFetch(
		fullRestaurantCacheKey,
		useCache,
		func() ([]maps.Restaurant, error) {
			// Get restaurant data using exportRestaurantData
			restaurantData, err := exportRestaurantData(subreddit, numPosts, useCache)
			if err != nil {
				return nil, err
			}

			// Create a Maps client for place ID lookups
			ctx := context.Background()
			mapsClient, err := maps.NewClient(ctx, cfg.GoogleMapsAPIKey)
			if err != nil {
				return nil, fmt.Errorf("error creating Maps client: %v", err)
			}
			defer mapsClient.Close()

			// Process each restaurant to add/canonicalize Google Maps links
			var fullRestaurants []maps.Restaurant
			for _, restaurant := range restaurantData {
				result, err := mapsClient.FetchGoogleMapsLink(ctx, &restaurant, mapsQueryHint)
				if err != nil {
					fmt.Printf("Warning: error fetching Maps link for %s: %v\n", restaurant.Name, err)
					continue
				}
				if result != nil {
					fullRestaurants = append(fullRestaurants, *result)
				}
				// Add 2 second delay between API calls
				time.Sleep(2 * time.Second)
			}

			fmt.Printf("Successfully exported %d restaurants with Maps data from r/%s\n", len(fullRestaurants), subreddit)
			return fullRestaurants, nil
		},
	)
}

// exportToCSV exports restaurant data to a CSV file
func exportToCSV(subreddit string, numPosts int, useCache bool) error {
	// Get the full restaurant data
	restaurants, err := exportFullRestaurantData(subreddit, numPosts, useCache)
	if err != nil {
		return fmt.Errorf("error getting restaurant data: %v", err)
	}

	// Sort restaurants by upvotes in descending order
	sort.Slice(restaurants, func(i, j int) bool {
		return restaurants[i].Upvotes > restaurants[j].Upvotes
	})

	// Apply numOutput limit if specified
	if numOutput > 0 && len(restaurants) > numOutput {
		restaurants = restaurants[:numOutput]
	}

	// Create CSV filename with date and time range
	currentDate := time.Now().Format("20060102")
	filename := fmt.Sprintf("%s_%s_%s.csv", subreddit, currentDate, timeRange)

	// Create CSV writer
	writer, err := csv.NewWriter(filename)
	if err != nil {
		return fmt.Errorf("error creating CSV writer: %v", err)
	}
	defer writer.Close()

	// Write header
	header := []string{"Name", "Type", "Google Maps url", "Google Maps rating", "Reddit url", "Lat", "Lng"}
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing CSV header: %v", err)
	}

	// Write data rows
	for i, restaurant := range restaurants {
		row := []string{
			fmt.Sprintf("%s (#%d, %d upvotes)", restaurant.GoogleMapsData.Name, i+1, restaurant.Upvotes),
			restaurant.GoogleMapsData.Type,
			restaurant.GoogleMapsData.GoogleMapsUrl,
			fmt.Sprintf("%.1f (%d reviews)", restaurant.GoogleMapsData.Rating, restaurant.GoogleMapsData.UserRatingCount),
			restaurant.RedditUrl,
			fmt.Sprintf("%.6f", restaurant.GoogleMapsData.Latitude),
			fmt.Sprintf("%.6f", restaurant.GoogleMapsData.Longitude),
		}
		if err := writer.WriteRow(row); err != nil {
			return fmt.Errorf("error writing CSV row: %v", err)
		}
	}

	fmt.Printf("Successfully exported %d restaurants to %s\n", len(restaurants), writer.Path())
	return nil
}
