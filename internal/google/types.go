package google

import (
    "net/http" 
)

type searchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

// Client wraps the Google Custom Search API for typo correction.
type Client struct {
	apiKey string
	cx     string
	httpClient *http.Client
}