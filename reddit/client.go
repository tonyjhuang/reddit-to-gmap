package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	baseURL     = "https://oauth.reddit.com"
	tokenURL    = "https://www.reddit.com/api/v1/access_token"
	placeholderClientID = "YOUR_CLIENT_ID"
	placeholderClientSecret = "YOUR_CLIENT_SECRET"
)

type Client struct {
	httpClient *http.Client
	token      string
	clientID   string
	clientSecret string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Post struct {
	Data struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		Permalink string `json:"permalink"`
		Selftext  string `json:"selftext"`  // Description/body of the post
		Score     int    `json:"score"`     // Number of upvotes
		// Add more fields as needed
	} `json:"data"`
}

type ListingResponse struct {
	Data struct {
		Children []Post `json:"children"`
	} `json:"data"`
}

func getEnvVar(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
		clientID: getEnvVar("REDDIT_CLIENT_ID", placeholderClientID),
		clientSecret: getEnvVar("REDDIT_CLIENT_SECRET", placeholderClientSecret),
	}
}

func (c *Client) getToken() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error getting token: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("error decoding token response: %v", err)
	}

	c.token = tokenResp.AccessToken
	return nil
}

func (c *Client) GetPosts(subreddit string, limit int) ([]Post, error) {
	if c.token == "" {
		if err := c.getToken(); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("%s/r/%s/hot?limit=%d", baseURL, subreddit, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("User-Agent", "reddit-to-gmap/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting posts: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var listingResp ListingResponse
	if err := json.Unmarshal(body, &listingResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return listingResp.Data.Children, nil
} 