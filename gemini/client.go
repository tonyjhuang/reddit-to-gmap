package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/tonyjhuang/reddit-to-gmap/reddit"
	"google.golang.org/api/option"
)

type Restaurant struct {
	Name          string `json:"name"`
	Upvotes       int    `json:"upvotes"`
	RedditUrl     string `json:"reddit_url"`
	Neighborhood  string `json:"neighborhood,omitempty"`
	GoogleMapsUrl string `json:"google_maps_url,omitempty"`
}

type Client struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewClient(ctx context.Context) (*Client, error) {
	apiKey := os.Getenv("GOOGLE_GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_GEMINI_API_KEY environment variable is required")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	model := client.GenerativeModel("gemini-2.0-flash-lite")

	model.SetTemperature(0)
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"restaurants": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"name", "upvotes", "reddit_url"},
					Properties: map[string]*genai.Schema{
						"name":         {Type: genai.TypeString},
						"upvotes":      {Type: genai.TypeInteger},
						"reddit_url":   {Type: genai.TypeString},
						"neighborhood": {Type: genai.TypeString},
					},
				},
			},
		},
	}

	return &Client{
		client: client,
		model:  model,
	}, nil
}

func (c *Client) Close() {
	c.client.Close()
}

// ToRestaurantData processes Reddit posts and returns a slice of restaurants.
// Each restaurant corresponds to a Reddit post that was identified as a restaurant review.
func (c *Client) ToRestaurantData(ctx context.Context, posts []reddit.Post) ([]Restaurant, error) {
	// Convert posts to JSON for the prompt
	postsJSON, err := json.Marshal(posts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal posts: %v", err)
	}

	prompt := fmt.Sprintf(`
Each input object represents a Reddit post with title, description (selftext), etc., from a food subreddit. For each Reddit post that corresponds to a single restaurant review, transform it into a corresponding entry in the output.

A post is considered a restaurant review if the title mentions a specific restaurant name and the selftext contains details about the dining experience (e.g., food descriptions, reviews, prices). If the title contains the word 'review', 'recommendation', or 'ate at', consider it a restaurant review.

Skip any input Reddit posts that either don't correspond to a restaurant review or that appear to mention a list of restaurants. If a post's restaurant association is unclear, skip it.

Input posts:
%s`, string(postsJSON))

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	// Parse the response into our temporary struct
	var result struct {
		Restaurants []Restaurant `json:"restaurants"`
	}
	if err := json.Unmarshal([]byte(resp.Candidates[0].Content.Parts[0].(genai.Text)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Restaurants, nil
}
