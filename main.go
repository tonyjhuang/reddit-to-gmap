package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tonyjhuang/reddit-to-gmap/cache"
	"github.com/tonyjhuang/reddit-to-gmap/reddit"
)

var (
	subreddit string
	numPosts  int
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
		return exportReddit(subreddit, numPosts)
	},
}

var printGmapLinksCmd = &cobra.Command{
	Use:   "print-gmap-links",
	Short: "Print Google Maps links from cached Reddit posts",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printGmapLinks(subreddit, numPosts)
	},
}

func init() {
	rootCmd.AddCommand(exportRedditCmd)
	rootCmd.AddCommand(printGmapLinksCmd)

	// Add flags to both commands
	for _, cmd := range []*cobra.Command{exportRedditCmd, printGmapLinksCmd} {
		cmd.Flags().StringVarP(&subreddit, "subreddit", "s", "", "Subreddit to fetch posts from (required)")
		cmd.Flags().IntVarP(&numPosts, "num-posts", "n", 10, "Number of posts to fetch")
		cmd.MarkFlagRequired("subreddit")
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func exportReddit(subreddit string, numPosts int) error {
	client := reddit.NewClient()
	posts, err := client.GetPosts(subreddit, numPosts)
	if err != nil {
		return fmt.Errorf("error fetching posts: %v", err)
	}

	if err := cache.WriteToCache(subreddit, posts); err != nil {
		return fmt.Errorf("error writing to cache: %v", err)
	}

	fmt.Printf("Successfully exported %d posts from r/%s\n", len(posts), subreddit)
	return nil
}

func printGmapLinks(subreddit string, numPosts int) error {
	var posts []reddit.Post

	if cache.CacheExists(subreddit) {
		cacheData, err := cache.ReadFromCache(subreddit)
		if err != nil {
			return fmt.Errorf("error reading from cache: %v", err)
		}
		posts = cacheData.Posts.([]reddit.Post)
	} else {
		// If cache doesn't exist, fetch from Reddit
		if err := exportReddit(subreddit, numPosts); err != nil {
			return err
		}
		cacheData, err := cache.ReadFromCache(subreddit)
		if err != nil {
			return fmt.Errorf("error reading from cache: %v", err)
		}
		posts = cacheData.Posts.([]reddit.Post)
	}

	// TODO: Implement Google Maps link generation
	fmt.Printf("Found %d posts from r/%s\n", len(posts), subreddit)
	fmt.Println("Google Maps link generation not yet implemented")
	return nil
} 