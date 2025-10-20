package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// RxNormCache stores RxNorm lookups so we avoid redundant API calls.
type RxNormCache struct {
	nameToRxCUI       map[string]RxCUIMatch
	rxcuiToTTY        map[string]string
	rxcuiToIngredient map[string]IngredientInfo
	path              string
}

// RxCUIMatch represents an RxCUI lookup result.
type RxCUIMatch struct {
	RxCUI       string `json:"rxcui"`
	MatchedName string `json:"matched_name"`
	Score       string `json:"score"`
}

// IngredientInfo summarizes the ingredient tied to a brand RxCUI.
type IngredientInfo struct {
	RxCUI string `json:"rxcui"`
	Name  string `json:"name"`
}

// NewRxNormCache constructs an empty cache bound to the provided path.
func NewRxNormCache(path string) *RxNormCache {
	return &RxNormCache{
		nameToRxCUI:       make(map[string]RxCUIMatch),
		rxcuiToTTY:        make(map[string]string),
		rxcuiToIngredient: make(map[string]IngredientInfo),
		path:              path,
	}
}

// Load populates the cache from disk when the file exists.
func (rc *RxNormCache) Load() error {
	data, err := os.ReadFile(rc.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read rxnorm cache: %w", err)
	}

	var payload struct {
		NameToRxCUI       map[string]RxCUIMatch     `json:"name_to_rxcui"`
		RxCUIToTTY        map[string]string         `json:"rxcui_to_tty"`
		RxCUIToIngredient map[string]IngredientInfo `json:"rxcui_to_ingredient"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse rxnorm cache: %w", err)
	}

	if payload.NameToRxCUI == nil {
		payload.NameToRxCUI = make(map[string]RxCUIMatch)
	}
	if payload.RxCUIToTTY == nil {
		payload.RxCUIToTTY = make(map[string]string)
	}
	if payload.RxCUIToIngredient == nil {
		payload.RxCUIToIngredient = make(map[string]IngredientInfo)
	}

	rc.nameToRxCUI = payload.NameToRxCUI
	rc.rxcuiToTTY = payload.RxCUIToTTY
	rc.rxcuiToIngredient = payload.RxCUIToIngredient

	return nil
}

// Save writes the cache contents to disk.
func (rc *RxNormCache) Save() error {
	payload := struct {
		NameToRxCUI       map[string]RxCUIMatch     `json:"name_to_rxcui"`
		RxCUIToTTY        map[string]string         `json:"rxcui_to_tty"`
		RxCUIToIngredient map[string]IngredientInfo `json:"rxcui_to_ingredient"`
	}{
		NameToRxCUI:       rc.nameToRxCUI,
		RxCUIToTTY:        rc.rxcuiToTTY,
		RxCUIToIngredient: rc.rxcuiToIngredient,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rxnorm cache: %w", err)
	}

	if err := os.WriteFile(rc.path, data, 0o644); err != nil {
		return fmt.Errorf("write rxnorm cache: %w", err)
	}

	return nil
}

// GetRxCUI resolves a medication name to the cached RxCUI metadata.
func (rc *RxNormCache) GetRxCUI(name string) (RxCUIMatch, bool) {
	if match, ok := rc.nameToRxCUI[name]; ok {
		return match, true
	}

	lower := strings.ToLower(name)
	for k, v := range rc.nameToRxCUI {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}

	return RxCUIMatch{}, false
}

// SetRxCUI caches an RxCUI match for the supplied name.
func (rc *RxNormCache) SetRxCUI(name string, match RxCUIMatch) {
	rc.nameToRxCUI[name] = match
}

// GetTTY retrieves a cached TTY value for the RxCUI.
func (rc *RxNormCache) GetTTY(rxcui string) (string, bool) {
	tty, ok := rc.rxcuiToTTY[rxcui]
	return tty, ok
}

// SetTTY stores the TTY for the RxCUI.
func (rc *RxNormCache) SetTTY(rxcui, tty string) {
	rc.rxcuiToTTY[rxcui] = tty
}

// GetIngredient retrieves a cached ingredient for the RxCUI.
func (rc *RxNormCache) GetIngredient(rxcui string) (IngredientInfo, bool) {
	ingredient, ok := rc.rxcuiToIngredient[rxcui]
	return ingredient, ok
}

// SetIngredient caches ingredient data for the RxCUI.
func (rc *RxNormCache) SetIngredient(rxcui string, info IngredientInfo) {
	rc.rxcuiToIngredient[rxcui] = info
}

// Stats summarizes cache sizes for logging purposes.
func (rc *RxNormCache) Stats() (nameCount, ttyCount, ingredientCount int) {
	return len(rc.nameToRxCUI), len(rc.rxcuiToTTY), len(rc.rxcuiToIngredient)
}
