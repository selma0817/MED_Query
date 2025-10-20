package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	// DefaultFile is the config file that will be loaded when no explicit path is supplied.
	DefaultFile = "config.json"

	// DefaultTypoCacheFile and DefaultRxNormCacheFile are the on-disk caches used by the app.
	DefaultTypoCacheFile   = "typo_corrections_cache.json"
	DefaultRxNormCacheFile = "rxnorm_cache.json"

	// DefaultInputFile and DefaultOutputFile describe the Excel files used by the CLI.
	DefaultInputFile  = "inputs/all_med_med.xlsx"
	DefaultOutputFile = "output_results.xlsx"

	// DefaultAPIDelay throttles external API calls.
	DefaultAPIDelay = 200 * time.Millisecond
)

// Config represents optional runtime configuration sourced from config.json.
type Config struct {
	GoogleAPIKey         string `json:"google_api_key"`
	GoogleSearchEngineID string `json:"google_search_engine_id"`
}

// Load attempts to read and parse the JSON configuration at path.
// If the file does not exist, os.ErrNotExist is returned so callers can decide how to proceed.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// GoogleEnabled reports whether Google typo correction can be used.
func (c *Config) GoogleEnabled() bool {
	if c == nil {
		return false
	}
	return c.GoogleAPIKey != "" && c.GoogleSearchEngineID != ""
}
