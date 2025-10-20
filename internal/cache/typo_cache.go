package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TypoCache persists typo correction lookups across runs.
type TypoCache struct {
	corrections map[string]string
	path        string
}

// NewTypoCache constructs an empty cache tied to the supplied path.
func NewTypoCache(path string) *TypoCache {
	return &TypoCache{
		corrections: make(map[string]string),
		path:        path,
	}
}

// Load populates the cache from disk when the file exists.
func (tc *TypoCache) Load() error {
	data, err := os.ReadFile(tc.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read typo cache: %w", err)
	}

	if err := json.Unmarshal(data, &tc.corrections); err != nil {
		return fmt.Errorf("parse typo cache: %w", err)
	}

	return nil
}

// Save writes the current cache contents to disk.
func (tc *TypoCache) Save() error {
	data, err := json.MarshalIndent(tc.corrections, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal typo cache: %w", err)
	}

	if err := os.WriteFile(tc.path, data, 0o644); err != nil {
		return fmt.Errorf("write typo cache: %w", err)
	}

	return nil
}

// Get returns the corrected name for a typo, performing a case-insensitive lookup.
func (tc *TypoCache) Get(typo string) (string, bool) {
	if corrected, ok := tc.corrections[typo]; ok {
		return corrected, true
	}

	typoLower := strings.ToLower(typo)
	for k, v := range tc.corrections {
		if strings.ToLower(k) == typoLower {
			return v, true
		}
	}

	return "", false
}

// Set records a corrected name for the supplied typo.
func (tc *TypoCache) Set(typo, corrected string) {
	tc.corrections[typo] = corrected
}

// Size reports the number of cached typo corrections.
func (tc *TypoCache) Size() int {
	return len(tc.corrections)
}
