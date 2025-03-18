package maps

import (
	"context"
	"fmt"
	"os"
	"strings"

	places "cloud.google.com/go/maps/places/apiv1"
	placespb "cloud.google.com/go/maps/places/apiv1/placespb"
	"github.com/googleapis/gax-go/v2/callctx"
	"github.com/tonyjhuang/reddit-to-gmap/gemini"
	"google.golang.org/api/option"
)

type Client struct {
	client *places.Client
}

func NewClient(ctx context.Context) (*Client, error) {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_MAPS_API_KEY environment variable is required")
	}

	c, err := places.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Places client: %v", err)
	}

	return &Client{
		client: c,
	}, nil
}

func (c *Client) Close() {
	c.client.Close()
}

// FetchGoogleMapsLink processes a restaurant to either canonicalize its existing Google Maps link
// or search for a new one if none exists. For searches, it uses the restaurant name and neighborhood
// (if available) to find the most relevant match in NYC.
func (c *Client) FetchGoogleMapsLink(ctx context.Context, restaurant *gemini.Restaurant) error {
	fmt.Printf("Fetching Google Maps link for %s\n", restaurant.Name)

	// Build search query with restaurant name and location context
	query := restaurant.Name
	if restaurant.Neighborhood != "" {
		query = fmt.Sprintf("%s %s", query, restaurant.Neighborhood)
	}
	query = fmt.Sprintf("%s NYC", query)

	// Search for the place using Places API Text Search
	req := &placespb.SearchTextRequest{
		TextQuery: query,
	}

	// Set the required field mask header for all Places API requests
	ctx = callctx.SetHeaders(ctx, callctx.XGoogFieldMaskHeader, "*")

	resp, err := c.client.SearchText(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to search for place: %v", err)
	}

	if len(resp.Places) == 0 {
		fmt.Printf("No results found for %s: %s\n", restaurant.Name, query)
		return nil // No results found, leave GoogleMapsLink empty
	}

	// Get the first result's place ID and format as a Google Maps link
	placeID := strings.TrimPrefix(resp.Places[0].Name, "places/")
	restaurant.GoogleMapsUrl = fmt.Sprintf("https://www.google.com/maps/place/?q=place_id:%s", placeID)
	return nil
}
