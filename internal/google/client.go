package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 10 * time.Second


// NewClient constructs a Client ready to call the Custom Search endpoint.
func NewClient(httpClient *http.Client, apiKey, searchEngineID string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		apiKey:     apiKey,
		cx:         searchEngineID,
		httpClient: httpClient,
	}
}

// CorrectMedicationName attempts to infer the proper medication name using Google search.
func (c *Client) CorrectMedicationName(ctx context.Context, medName string) (string, error) {
	if medName == "" {
		return "", fmt.Errorf("google: empty medication name")
	}

	query := fmt.Sprintf("%s medication", medName)
	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s",
		url.QueryEscape(c.apiKey),
		url.QueryEscape(c.cx),
		url.QueryEscape(query),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("google: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("google: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("google: read response: %w", err)
	}

	var payload searchResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("google: parse response: %w", err)
	}

	if len(payload.Items) == 0 {
		return "", fmt.Errorf("google: no search results for %q", medName)
	}

	title := payload.Items[0].Title
	return extractMedicationName(title, medName), nil
}

func extractMedicationName(title, original string) string {
	separators := []string{" - ", ": ", " (", " | "}

	for _, sep := range separators {
		if idx := strings.Index(title, sep); idx > 0 {
			candidate := strings.TrimSpace(title[:idx])
			candidate = strings.Trim(candidate, ".,!?\"'")
			if len(candidate) > 0 && len(candidate) < 50 {
				return candidate
			}
		}
	}

	words := strings.Fields(title)
	if len(words) > 0 {
		return strings.Trim(words[0], ".,!?\"'")
	}

	return original
}

