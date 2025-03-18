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
5. Set up your Google Gemini API key:
   - Go to https://makersuite.google.com/app/apikey
   - Create a new API key
   - Set it as an environment variable:
     ```bash
     export GOOGLE_GEMINI_API_KEY="your_gemini_api_key"
     ```
6. Set up your Google Maps API key:
   - Go to https://console.cloud.google.com/
   - Create a new project or select an existing one
   - Enable the Places API
   - Go to Credentials
   - Click "Create Credentials" and select "API key"
   - Restrict the API key to only the Place Text Search API
   - Set it as an environment variable:
     ```bash
     export GOOGLE_MAPS_API_KEY="your_maps_api_key"
     ```

## Usage

The tool provides several commands to help you extract and process Reddit posts:

### Commands

#### Generate Top Post Google Map CSV

```bash
./reddit-to-gmap generate-top-post-google-map-csv --subreddit <subreddit> --num-posts <number> [--use-cache]
```

This command will:
1. Fetch the specified number of posts from the given subreddit
2. Process the posts to extract restaurant data
3. Generate a CSV file with restaurant information in the `csv/` directory
4. Use cached data if `--use-cache` is specified

## Flags

- `--subreddit, -s`: The subreddit to fetch posts from (required)
- `--num-posts, -n`: Number of posts to fetch (default: 10)
- `--use-cache`: Use cached data instead of fetching from Reddit

## Environment Variables

The following environment variables are required:

- `REDDIT_CLIENT_ID`: Your Reddit API client ID
- `REDDIT_CLIENT_SECRET`: Your Reddit API client secret
- `GOOGLE_GEMINI_API_KEY`: Your Google API key for Gemini
- `GOOGLE_MAPS_API_KEY`: Your Google API key for Maps and Places APIs

You can set these either:
1. In your shell:
   ```bash
   export REDDIT_CLIENT_ID="your_client_id"
   export REDDIT_CLIENT_SECRET="your_client_secret"
   export GOOGLE_GEMINI_API_KEY="your_gemini_api_key"
   export GOOGLE_MAPS_API_KEY="your_maps_api_key"
   ```

2. Or inline with the command:
   ```bash
   REDDIT_CLIENT_ID="your_client_id" REDDIT_CLIENT_SECRET="your_client_secret" GOOGLE_GEMINI_API_KEY="your_gemini_api_key" GOOGLE_MAPS_API_KEY="your_maps_api_key" ./reddit-to-gmap debug:export-reddit -s askreddit
   ```

## Output Files

The tool generates several types of output files:

- `cache/*.json`: Raw Reddit posts and processed restaurant data
- `csv/*.csv`: CSV files containing restaurant information
- `maps/*.json`: Processed restaurant data with Google Maps links


## Debug Commands

These commands are primarily for development and debugging purposes:

#### Debug: Export Reddit Posts

```bash
./reddit-to-gmap debug:export-reddit --subreddit <subreddit> --num-posts <number> [--use-cache]
```

This command will:
1. Fetch the specified number of posts from the given subreddit
2. Save them to a JSON file in the `cache/` directory
3. Use cached data if `--use-cache` is specified

#### Debug: Export Basic Restaurant Data

```bash
./reddit-to-gmap debug:export-restaurant-data --subreddit <subreddit> --num-posts <number> [--use-cache]
```

This command will:
1. Fetch the specified number of posts from the given subreddit
2. Use Google's Gemini AI to extract structured restaurant data from the posts
3. Save the restaurant data to a JSON file in the `cache/` directory
4. Use cached data if `--use-cache` is specified

#### Debug: Export Full Restaurant Data with Maps Links

```bash
./reddit-to-gmap debug:export-full-restaurant-data --subreddit <subreddit> --num-posts <number> [--use-cache]
```

This command will:
1. Fetch the specified number of posts from the given subreddit
2. Use Google's Gemini AI to extract structured restaurant data
3. Canonicalize restaurant names and generate Google Maps links
4. Save the complete data to a JSON file in the `cache/` directory
5. Use cached data if `--use-cache` is specified
