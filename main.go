package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonyjhuang/reddit-to-gmap/cache"
	"github.com/tonyjhuang/reddit-to-gmap/csv"
	"github.com/tonyjhuang/reddit-to-gmap/gemini"
	"github.com/tonyjhuang/reddit-to-gmap/maps"
	"github.com/tonyjhuang/reddit-to-gmap/reddit"
)

var (
	subreddit string
	numPosts  int
	useCache  bool
)

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
		cmd.MarkFlagRequired("subreddit")
	}

	// Add use-cache flag to export commands
	for _, cmd := range []*cobra.Command{exportRedditCmd, exportRestaurantDataCmd, exportFullRestaurantDataCmd, generateTopPostGoogleMapCSVCmd} {
		cmd.Flags().BoolVar(&useCache, "use-cache", true, "Whether to use cached data if available")
	}
}

func main() {
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
			client := reddit.NewClient()
			posts, err := client.GetPosts(subreddit, numPosts)
			if err != nil {
				return nil, fmt.Errorf("error fetching posts: %v", err)
			}
			fmt.Printf("Successfully exported %d posts from r/%s\n", len(posts), subreddit)
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
			// Get Reddit posts using exportReddit
			posts, err := exportReddit(subreddit, numPosts, useCache)
			if err != nil {
				return nil, err
			}

			// Create a Gemini client
			ctx := context.Background()
			geminiClient, err := gemini.NewClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("error creating Gemini client: %v", err)
			}
			defer geminiClient.Close()

			// Process the posts with Gemini
			restaurantData, err := geminiClient.ToRestaurantData(ctx, posts)
			if err != nil {
				return nil, fmt.Errorf("error processing posts with Gemini: %v", err)
			}

			fmt.Printf("Successfully exported %d restaurants from r/%s\n", len(restaurantData), subreddit)
			return restaurantData, nil
		},
	)
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
			mapsClient, err := maps.NewClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("error creating Maps client: %v", err)
			}
			defer mapsClient.Close()

			// Process each restaurant to add/canonicalize Google Maps links
			var fullRestaurants []maps.Restaurant
			for _, restaurant := range restaurantData {
				result, err := mapsClient.FetchGoogleMapsLink(ctx, &restaurant)
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

	// Create CSV writer
	writer, err := csv.NewWriter(fmt.Sprintf("%s.csv", subreddit))
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
			fmt.Sprintf("%s (#%d, %d upvotes)", restaurant.Name, i+1, restaurant.Upvotes),
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
