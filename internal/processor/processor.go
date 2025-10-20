package processor

import (
	"context"
	"fmt"
	"strings"
	"time"
	"fuzzyMed/internal/cache"
	"fuzzyMed/internal/google"
	"fuzzyMed/internal/rxnorm"
)


// New constructs a Processor with the supplied dependencies.
func New(
	typoCache *cache.TypoCache,
	rxnormCache *cache.RxNormCache,
	rxClient *rxnorm.Client,
	googleClient *google.Client,
	delay time.Duration,
	logger Logger,
) *Processor {
	if logger == nil {
		logger = func(string, ...any) {}
	}
	return &Processor{
		typoCache:   typoCache,
		rxnormCache: rxnormCache,
		rxClient:    rxClient,
		google:      googleClient,
		delay:       delay,
		logf:        logger,
	}
}

// ProcessMedications sequentially processes the supplied medications.
func (p *Processor) ProcessMedications(ctx context.Context, medications []string) []Result {
	results := make([]Result, 0, len(medications))
	for idx, med := range medications {
		p.logf("[%d/%d] Processing %s", idx+1, len(medications), med)
		results = append(results, p.ProcessMedication(ctx, med))
	}
	return results
}

// ProcessMedication handles a single medication lookup.
func (p *Processor) ProcessMedication(ctx context.Context, medName string) Result {
	result := Result{
		OriginalName: medName,
	}

	nameToTry := medName
	if corrected, ok := p.typoCache.Get(medName); ok {
		p.logf("  typo cache hit: %s -> %s", medName, corrected)
		nameToTry = corrected
		result.CorrectedName = corrected
		result.WasTypoCorrected = true
	}

	match, err := p.lookupRxCUI(ctx, nameToTry)
	if err != nil {
		p.logf("  no match: %v", err)
		if !result.WasTypoCorrected && p.google != nil {
			p.logf("  attempting Google typo correction...")
			if corrected, googleErr := p.google.CorrectMedicationName(ctx, medName); googleErr == nil && corrected != "" && !equalFold(corrected, medName) {
				p.logf("  Google correction: %s -> %s", medName, corrected)
				nameToTry = corrected
				result.CorrectedName = corrected
				result.WasTypoCorrected = true
				p.typoCache.Set(medName, corrected)
				match, err = p.lookupRxCUI(ctx, corrected)
			} else if googleErr != nil {
				p.logf("  Google error: %v", googleErr)
			}
		}
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.HasMatch = true
	result.MatchedRxCUI = match.RxCUI
	result.MatchedName = match.Name
	result.Score = match.Score
	p.logf("  matched %s (RxCUI %s, score %s)", match.Name, match.RxCUI, match.Score)
	p.sleep()

	tty, err := p.lookupTTY(ctx, match.RxCUI)
	if err != nil {
		result.Error = fmt.Sprintf("TTY error: %v", err)
		p.logf("  TTY error: %v", err)
		return result
	}

	result.TTY = tty
	result.TTYDescription = rxnorm.DescribeTTY(tty)
	p.logf("  tty %s (%s)", tty, result.TTYDescription)

	if tty != "IN" {
		ingredient, ingredientErr := p.lookupIngredient(ctx, match.RxCUI)
		if ingredientErr != nil {
			p.logf("  no ingredient: %v", ingredientErr)
		} else {
			result.IngredientRxCUI = ingredient.RxCUI
			result.IngredientName = ingredient.Name
			p.logf("  ingredient %s (RxCUI %s)", ingredient.Name, ingredient.RxCUI)
		}
	}

	return result
}

func (p *Processor) lookupRxCUI(ctx context.Context, medName string) (rxnorm.Match, error) {
	if match, ok := p.rxnormCache.GetRxCUI(medName); ok {
		return rxnorm.Match{
			RxCUI: match.RxCUI,
			Name:  match.MatchedName,
			Score: match.Score,
		}, nil
	}

	match, err := p.rxClient.ApproximateMatch(ctx, medName)
	if err != nil {
		return rxnorm.Match{}, err
	}

	cacheMatch := cache.RxCUIMatch{
		RxCUI:       match.RxCUI,
		MatchedName: match.Name,
		Score:       match.Score,
	}
	p.rxnormCache.SetRxCUI(medName, cacheMatch)
	p.rxnormCache.SetRxCUI(match.Name, cacheMatch)

	p.sleep()
	return match, nil
}

func (p *Processor) lookupTTY(ctx context.Context, rxcui string) (string, error) {
	if tty, ok := p.rxnormCache.GetTTY(rxcui); ok {
		return tty, nil
	}

	tty, err := p.rxClient.TTY(ctx, rxcui)
	if err != nil {
		return "", err
	}

	p.rxnormCache.SetTTY(rxcui, tty)
	p.sleep()
	return tty, nil
}

func (p *Processor) lookupIngredient(ctx context.Context, rxcui string) (rxnorm.Ingredient, error) {
	if ingredient, ok := p.rxnormCache.GetIngredient(rxcui); ok {
		return rxnorm.Ingredient{
			RxCUI: ingredient.RxCUI,
			Name:  ingredient.Name,
		}, nil
	}

	ingredient, err := p.rxClient.IngredientFor(ctx, rxcui)
	if err != nil {
		return rxnorm.Ingredient{}, err
	}

	p.rxnormCache.SetIngredient(rxcui, cache.IngredientInfo{
		RxCUI: ingredient.RxCUI,
		Name:  ingredient.Name,
	})
	p.rxnormCache.SetRxCUI(ingredient.Name, cache.RxCUIMatch{
		RxCUI:       ingredient.RxCUI,
		MatchedName: ingredient.Name,
		Score:       "100",
	})

	p.sleep()
	return ingredient, nil
}

func (p *Processor) sleep() {
	if p.delay > 0 {
		time.Sleep(p.delay)
	}
}

// BuildMaps converts results into the aggregated maps used by downstream processors.
func BuildMaps(results []Result) Maps {
	maps := Maps{
		NameToRxCUI:       make(map[string]string),
		BrandToIngredient: make(map[string]string),
		TypoToCorrect:     make(map[string]string),
		NoMatch:           []string{},
	}

	for _, result := range results {
		if !result.HasMatch {
			maps.NoMatch = append(maps.NoMatch, result.OriginalName)
			continue
		}

		maps.NameToRxCUI[result.MatchedName] = result.MatchedRxCUI
		if result.IngredientName != "" && result.IngredientRxCUI != "" {
			maps.NameToRxCUI[result.IngredientName] = result.IngredientRxCUI
		}

		if result.TTY != "IN" && result.IngredientName != "" {
			maps.BrandToIngredient[result.MatchedName] = result.IngredientName
		}

		if result.WasTypoCorrected && result.CorrectedName != "" {
			maps.TypoToCorrect[result.OriginalName] = result.CorrectedName
		}
	}

	return maps
}

func equalFold(a, b string) bool {
	return len(a) == len(b) && strings.EqualFold(a, b)
}
