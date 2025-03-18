# Reddit to Google Maps

A CLI tool that allows you to export Reddit posts and generate Google Maps links from location data.

## Setup

1. Install Go 1.24.1 or later
2. Clone this repository
3. Run `go mod tidy` to install dependencies
4. Set up your Reddit API credentials:
   - Go to https://www.reddit.com/prefs/apps
   - Click "create another app..."
   - Choose "script"
   - Fill in the required information
   - Once created, you'll get a client ID and client secret
   - Set these as environment variables:
     ```bash
     export REDDIT_CLIENT_ID="your_client_id"
     export REDDIT_CLIENT_SECRET="your_client_secret"
     ```
5. Set up your Google API key:
   - Go to https://makersuite.google.com/app/apikey
   - Create a new API key
   - Set it as an environment variable:
     ```bash
     export GOOGLE_API_KEY="your_google_api_key"
     ```

## Usage

The tool provides two main commands:

### Export Reddit Posts

```bash
./reddit-to-gmap export-reddit --subreddit <subreddit> --num-posts <number>
```

This command will:
1. Fetch the specified number of posts from the given subreddit
2. Save them to a JSON file in the `cache/` directory

### Export Restaurant Data

```bash
./reddit-to-gmap export-restaurant-data --subreddit <subreddit> --num-posts <number>
```

This command will:
1. Fetch the specified number of posts from the given subreddit
2. Use Google's Gemini AI to extract structured restaurant data from the posts
3. Save the restaurant data to a JSON file in the `cache/` directory

### Print Google Maps Links

```bash
./reddit-to-gmap print-gmap-links --subreddit <subreddit> --num-posts <number>
```

This command will:
1. Check if there's a cached file for the given subreddit
2. If not, fetch posts from Reddit and cache them
3. Read the cached posts and generate Google Maps links (TODO)

## Flags

- `--subreddit, -s`: The subreddit to fetch posts from (required)
- `--num-posts, -n`: Number of posts to fetch (default: 10)

## Environment Variables

The following environment variables are required:

- `REDDIT_CLIENT_ID`: Your Reddit API client ID
- `REDDIT_CLIENT_SECRET`: Your Reddit API client secret
- `GOOGLE_API_KEY`: Your Google API key for Gemini

You can set these either:
1. In your shell:
   ```bash
   export REDDIT_CLIENT_ID="your_client_id"
   export REDDIT_CLIENT_SECRET="your_client_secret"
   export GOOGLE_API_KEY="your_google_api_key"
   ```

2. Or inline with the command:
   ```bash
   REDDIT_CLIENT_ID="your_client_id" REDDIT_CLIENT_SECRET="your_client_secret" GOOGLE_API_KEY="your_google_api_key" ./reddit-to-gmap export-reddit -s askreddit
   ```