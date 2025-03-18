package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tonyjhuang/reddit-to-gmap/cache"
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
	Use:   "export-reddit",
	Short: "Export posts from a subreddit to a local cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exportReddit(subreddit, numPosts, useCache)
		return err
	},
}

var exportFullRestaurantDataCmd = &cobra.Command{
	Use:   "export-full-restaurant-data",
	Short: "Export restaurant data with canonicalized Google Maps links",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exportFullRestaurantData(subreddit, numPosts, useCache)
		return err
	},
}

var exportRestaurantDataCmd = &cobra.Command{
	Use:   "export-restaurant-data",
	Short: "Export restaurant data from Reddit posts using Gemini",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exportRestaurantData(subreddit, numPosts, useCache)
		return err
	},
}

func init() {
	rootCmd.AddCommand(exportRedditCmd)
	rootCmd.AddCommand(exportRestaurantDataCmd)
	rootCmd.AddCommand(exportFullRestaurantDataCmd)

	// Add flags to all commands
	for _, cmd := range []*cobra.Command{exportRedditCmd, exportRestaurantDataCmd, exportFullRestaurantDataCmd} {
		cmd.Flags().StringVarP(&subreddit, "subreddit", "s", "", "Subreddit to fetch posts from (required)")
		cmd.Flags().IntVarP(&numPosts, "num-posts", "n", 10, "Number of posts to fetch")
		cmd.MarkFlagRequired("subreddit")
	}

	// Add use-cache flag to export commands
	for _, cmd := range []*cobra.Command{exportRedditCmd, exportRestaurantDataCmd, exportFullRestaurantDataCmd} {
		cmd.Flags().BoolVar(&useCache, "use-cache", true, "Whether to use cached data if available")
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// exportReddit fetches Reddit posts and caches them. Returns the fetched posts.
func exportReddit(subreddit string, numPosts int, useCache bool) ([]reddit.Post, error) {
	var posts []reddit.Post

	// Check cache first if enabled
	if useCache && cache.CacheExists(subreddit) {
		cacheData, err := cache.ReadFromCache(subreddit)
		if err != nil {
			return nil, fmt.Errorf("error reading from cache: %v", err)
		}

		// Convert cached data back to []reddit.Post using JSON marshaling/unmarshaling
		jsonData, err := json.Marshal(cacheData.Data)
		if err != nil {
			return nil, fmt.Errorf("error marshaling cache data: %v", err)
		}

		if err := json.Unmarshal(jsonData, &posts); err != nil {
			return nil, fmt.Errorf("error unmarshaling cache data to posts: %v", err)
		}

		fmt.Printf("exportReddit: Found %d posts in cache for r/%s\n", len(posts), subreddit)
		return posts, nil
	}

	// Fetch fresh posts from Reddit
	client := reddit.NewClient()
	var err error
	posts, err = client.GetPosts(subreddit, numPosts)
	if err != nil {
		return nil, fmt.Errorf("error fetching posts: %v", err)
	}

	// Cache the posts
	if err := cache.WriteToCache(subreddit, posts); err != nil {
		return nil, fmt.Errorf("error writing to cache: %v", err)
	}

	fmt.Printf("Successfully exported %d posts from r/%s\n", len(posts), subreddit)
	return posts, nil
}

// exportRestaurantData processes Reddit posts into restaurant data and caches the results.
// Returns the processed restaurant data.
func exportRestaurantData(subreddit string, numPosts int, useCache bool) ([]gemini.Restaurant, error) {
	// Check if we already have processed restaurant data in cache
	restaurantCacheKey := subreddit + "_restaurants"
	if useCache && cache.CacheExists(restaurantCacheKey) {
		cacheData, err := cache.ReadFromCache(restaurantCacheKey)
		if err != nil {
			return nil, fmt.Errorf("error reading from cache: %v", err)
		}

		// Convert cached data back to []gemini.Restaurant using JSON marshaling/unmarshaling
		var restaurants []gemini.Restaurant
		jsonData, err := json.Marshal(cacheData.Data)
		if err != nil {
			return nil, fmt.Errorf("error marshaling cache data: %v", err)
		}

		if err := json.Unmarshal(jsonData, &restaurants); err != nil {
			return nil, fmt.Errorf("exportRestaurantData: error unmarshaling cache data to restaurants: %v", err)
		}

		fmt.Printf("exportRestaurantData: Found %d restaurants in cache for r/%s\n", len(restaurants), subreddit)
		return restaurants, nil
	}

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

	// Cache the results
	if err := cache.WriteToCache(restaurantCacheKey, restaurantData); err != nil {
		return nil, fmt.Errorf("error writing to cache: %v", err)
	}

	fmt.Printf("Successfully exported %d restaurants from r/%s\n", len(restaurantData), subreddit)
	return restaurantData, nil
}

// exportFullRestaurantData processes Reddit posts into restaurant data with canonicalized Google Maps links.
// Returns the processed restaurant data.
func exportFullRestaurantData(subreddit string, numPosts int, useCache bool) ([]gemini.Restaurant, error) {
	// Check if we already have processed full restaurant data in cache
	fullRestaurantCacheKey := subreddit + "_full_restaurants"
	if useCache && cache.CacheExists(fullRestaurantCacheKey) {
		cacheData, err := cache.ReadFromCache(fullRestaurantCacheKey)
		if err != nil {
			return nil, fmt.Errorf("error reading from cache: %v", err)
		}

		// Convert cached data back to []gemini.Restaurant using JSON marshaling/unmarshaling
		var restaurants []gemini.Restaurant
		jsonData, err := json.Marshal(cacheData.Data)
		if err != nil {
			return nil, fmt.Errorf("error marshaling cache data: %v", err)
		}

		if err := json.Unmarshal(jsonData, &restaurants); err != nil {
			return nil, fmt.Errorf("exportFullRestaurantData: error unmarshaling cache data to restaurants: %v", err)
		}

		fmt.Printf("exportFullRestaurantData: Found %d restaurants in cache for r/%s\n", len(restaurants), subreddit)
		return restaurants, nil
	}

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
	for i := range restaurantData {
		if err := mapsClient.FetchGoogleMapsLink(ctx, &restaurantData[i]); err != nil {
			fmt.Printf("Warning: error fetching Maps link for %s: %v\n", restaurantData[i].Name, err)
			continue
		}
	}

	// Cache the results
	if err := cache.WriteToCache(fullRestaurantCacheKey, restaurantData); err != nil {
		return nil, fmt.Errorf("error writing to cache: %v", err)
	}

	fmt.Printf("Successfully exported %d restaurants with Maps links from r/%s\n", len(restaurantData), subreddit)
	return restaurantData, nil
}
