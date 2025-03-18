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

type GoogleMapsData struct {
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Rating          float64 `json:"rating"`
	UserRatingCount int     `json:"user_rating_count"`
	GoogleMapsUrl   string  `json:"google_maps_url"`
	Type            string  `json:"type"`
}

type Restaurant struct {
	Name           string         `json:"name"`
	Upvotes        int            `json:"upvotes"`
	RedditUrl      string         `json:"reddit_url"`
	Neighborhood   string         `json:"neighborhood,omitempty"`
	GoogleMapsData GoogleMapsData `json:"google_maps_data"`
}

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
func (c *Client) FetchGoogleMapsLink(ctx context.Context, restaurant *gemini.Restaurant) (*Restaurant, error) {
	fmt.Printf("Fetching Google Maps data for %s\n", restaurant.Name)

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
		return nil, fmt.Errorf("failed to search for place: %v", err)
	}

	if len(resp.Places) == 0 {
		fmt.Printf("No results found for %s: %s\n", restaurant.Name, query)
		return nil, nil // No results found
	}

	// Get the first result's place ID and format as a Google Maps link
	place := resp.Places[0]
	placeID := strings.TrimPrefix(place.Name, "places/")

	if place.UserRatingCount == nil {
		fmt.Printf("No user rating count found for %s\n", restaurant.Name)
		fmt.Printf("Place: %+v\n", place)
		return nil, nil
	}

	// Create the new Restaurant struct with all the data
	result := &Restaurant{
		Name:         restaurant.Name,
		Upvotes:      restaurant.Upvotes,
		RedditUrl:    restaurant.RedditUrl,
		Neighborhood: restaurant.Neighborhood,
		GoogleMapsData: GoogleMapsData{
			Latitude:        place.Location.Latitude,
			Longitude:       place.Location.Longitude,
			Rating:          float64(place.Rating),
			UserRatingCount: int(*place.UserRatingCount),
			GoogleMapsUrl:   fmt.Sprintf("https://www.google.com/maps/place/?q=place_id:%s", placeID),
			Type:            place.PrimaryTypeDisplayName.Text,
		},
	}

	return result, nil
}
