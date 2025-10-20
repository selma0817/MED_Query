package rxnorm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// DefaultBaseURL is the public RxNorm REST endpoint.
	DefaultBaseURL = "https://rxnav.nlm.nih.gov/REST"
	// DefaultMinScore filters out noisy approximate matches.
	DefaultMinScore = 10
	defaultTimeout = 10 * time.Second
)


// Constructor for Client 
func NewClient(httpClient *http.Client, baseURL string, minScore int) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if minScore == 0 {
		minScore = DefaultMinScore
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		minScore:   minScore,
	}
}

// ApproximateMatch returns the best RxNorm candidate for the supplied medication name.
func (c *Client) ApproximateMatch(ctx context.Context, medName string) (Match, error) {
	encoded := url.QueryEscape(medName)
	endpoint := fmt.Sprintf("%s/approximateTerm.json?term=%s&maxEntries=10", c.baseURL, encoded)

	var payload approximateMatchResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return Match{}, err
	}

	for _, candidate := range payload.ApproximateGroup.Candidate {
		var scoreValue int
		fmt.Sscanf(candidate.Score, "%d", &scoreValue)
		if candidate.Source == "RXNORM" && scoreValue >= c.minScore {
			return Match{
				RxCUI: candidate.RxCUI,
				Name:  candidate.Name,
				Score: candidate.Score,
			}, nil
		}
	}

	return Match{}, fmt.Errorf("rxnorm: no match found for %q with score >= %d", medName, c.minScore)
}

// TTY fetches the RxNorm Term Type for the supplied RxCUI.
func (c *Client) TTY(ctx context.Context, rxcui string) (string, error) {
	endpoint := fmt.Sprintf("%s/rxcui/%s/allProperties.json?prop=ATTRIBUTES", c.baseURL, rxcui)

	var payload allPropertiesResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return "", err
	}

	for _, prop := range payload.PropConceptGroup.PropConcept {
		if prop.PropName == "TTY" {
			return prop.PropValue, nil
		}
	}

	return "", fmt.Errorf("rxnorm: no TTY information for RxCUI %s", rxcui)
}

// IngredientFor resolves the related ingredient for the supplied RxCUI (if any).
func (c *Client) IngredientFor(ctx context.Context, rxcui string) (Ingredient, error) {
	endpoint := fmt.Sprintf("%s/rxcui/%s/related.json?tty=IN", c.baseURL, rxcui)

	var payload relatedResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return Ingredient{}, err
	}

	for _, group := range payload.RelatedGroup.ConceptGroup {
		if group.TTY == "IN" && len(group.ConceptProperties) > 0 {
			first := group.ConceptProperties[0]
			return Ingredient{
				RxCUI: first.RxCUI,
				Name:  first.Name,
			}, nil
		}
	}

	return Ingredient{}, fmt.Errorf("rxnorm: no ingredient found for RxCUI %s", rxcui)
}

// DescribeTTY provides a friendly description for a TTY code.
func DescribeTTY(tty string) string {
	lookup := map[string]string{
		"BN":   "Brand Name",
		"IN":   "Ingredient (Generic)",
		"PIN":  "Precise Ingredient",
		"MIN":  "Multiple Ingredients",
		"SCD":  "Semantic Clinical Drug (Generic)",
		"SBD":  "Semantic Branded Drug",
		"GPCK": "Generic Pack",
		"BPCK": "Brand Name Pack",
		"SCDG": "Semantic Clinical Drug Group (Generic)",
		"SBDG": "Semantic Branded Drug Group",
		"SCDF": "Semantic Clinical Drug Form (Generic)",
		"SBDF": "Semantic Branded Drug Form",
	}

	if description, ok := lookup[tty]; ok {
		return description
	}
	return "Unknown Type"
}

func (c *Client) getJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("rxnorm: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("rxnorm: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("rxnorm: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("rxnorm: read response: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("rxnorm: parse response: %w", err)
	}

	return nil
}
