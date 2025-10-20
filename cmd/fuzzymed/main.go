package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"fuzzyMed/internal/cache"
	"fuzzyMed/internal/config"
	"fuzzyMed/internal/excel"
	"fuzzyMed/internal/google"
	"fuzzyMed/internal/processor"
	"fuzzyMed/internal/rxnorm"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fuzzymed: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	fmt.Println("========================================================================")
	fmt.Println("RxNorm Medication Matcher with Typo Correction")
	fmt.Println("========================================================================")
	fmt.Println()

	cfg, err := config.Load(config.DefaultFile)
	switch {
	case err == nil:
		fmt.Println("[ok] configuration loaded")
		if cfg.GoogleEnabled() {
			fmt.Println("[ok] Google Custom Search enabled")
		} else {
			fmt.Println("[!] Google credentials missing or incomplete; typo correction disabled")
		}
	case errors.Is(err, os.ErrNotExist):
		fmt.Printf("[!] no %s found; running without Google typo correction\n", config.DefaultFile)
	default:
		return fmt.Errorf("load config: %w", err)
	}
	fmt.Println()

	typoCache := cache.NewTypoCache(config.DefaultTypoCacheFile)
	if err := typoCache.Load(); err != nil {
		return fmt.Errorf("load typo cache: %w", err)
	}
	fmt.Printf("[ok] loaded typo cache (%d entries)\n", typoCache.Size())

	rxCache := cache.NewRxNormCache(config.DefaultRxNormCacheFile)
	if err := rxCache.Load(); err != nil {
		return fmt.Errorf("load RxNorm cache: %w", err)
	}
	nameCount, ttyCount, ingredientCount := rxCache.Stats()
	fmt.Printf("[ok] loaded RxNorm cache (%d names, %d tty, %d ingredients)\n", nameCount, ttyCount, ingredientCount)
	fmt.Println()

	medications, columns, err := excel.ReadMedications(config.DefaultInputFile)
	if err != nil {
		return fmt.Errorf("read medications: %w", err)
	}
	fmt.Printf("[ok] reading medications from %s\n", config.DefaultInputFile)
	fmt.Printf("[ok] detected %d medication columns: %s\n", len(columns), strings.Join(columns, ", "))
	fmt.Printf("[ok] found %d medications to process\n\n", len(medications))

	var googleClient *google.Client
	if cfg != nil && cfg.GoogleEnabled() {
		googleClient = google.NewClient(nil, cfg.GoogleAPIKey, cfg.GoogleSearchEngineID)
	}

	logger := func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	}

	proc := processor.New(
		typoCache,
		rxCache,
		rxnorm.NewClient(nil, rxnorm.DefaultBaseURL, rxnorm.DefaultMinScore),
		googleClient,
		config.DefaultAPIDelay,
		logger,
	)

	results := make([]processor.Result, 0, len(medications))
	for idx, med := range medications {
		result := proc.ProcessMedication(ctx, med)
		results = append(results, result)

		if (idx+1)%10 == 0 {
			if err := typoCache.Save(); err != nil {
				fmt.Printf("[warn] saving typo cache: %v\n", err)
			}
			if err := rxCache.Save(); err != nil {
				fmt.Printf("[warn] saving RxNorm cache: %v\n", err)
			}
		}
	}

	fmt.Println()
	fmt.Println("[ok] saving caches to disk")
	if err := typoCache.Save(); err != nil {
		return fmt.Errorf("save typo cache: %w", err)
	}
	if err := rxCache.Save(); err != nil {
		return fmt.Errorf("save RxNorm cache: %w", err)
	}

	maps := processor.BuildMaps(results)
	printMaps(maps)

	fmt.Printf("[ok] writing results to %s\n", config.DefaultOutputFile)
	if err := excel.WriteResults(config.DefaultOutputFile, results); err != nil {
		return fmt.Errorf("write results: %w", err)
	}

	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("Summary")
	fmt.Println("========================================================================")
	fmt.Printf("Total medications   : %d\n", len(medications))
	fmt.Printf("Matched             : %d\n", len(results)-len(maps.NoMatch))
	fmt.Printf("Typo corrected      : %d\n", len(maps.TypoToCorrect))
	fmt.Printf("No match            : %d\n", len(maps.NoMatch))
	fmt.Printf("Brand -> Ingredient : %d\n", len(maps.BrandToIngredient))
	fmt.Println("========================================================================")

	return nil
}

func printMaps(maps processor.Maps) {
	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("Map: Name -> RxCUI")
	fmt.Println("========================================================================")
	for name, rxcui := range maps.NameToRxCUI {
		fmt.Printf("  %s -> %s\n", name, rxcui)
	}

	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("Map: Brand Name -> Ingredient Name")
	fmt.Println("========================================================================")
	for brand, ingredient := range maps.BrandToIngredient {
		fmt.Printf("  %s -> %s\n", brand, ingredient)
	}

	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("Map: Typo -> Corrected Name")
	fmt.Println("========================================================================")
	if len(maps.TypoToCorrect) == 0 {
		fmt.Println("  (no typo corrections)")
	} else {
		for typo, corrected := range maps.TypoToCorrect {
			fmt.Printf("  %s -> %s\n", typo, corrected)
		}
	}

	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("No Match List")
	fmt.Println("========================================================================")
	if len(maps.NoMatch) == 0 {
		fmt.Println("  (all medications matched)")
	} else {
		for _, name := range maps.NoMatch {
			fmt.Printf("  %s\n", name)
		}
	}
	fmt.Println()
}
