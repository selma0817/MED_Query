package processor
import (
	"time"
	"fuzzyMed/internal/cache"
	"fuzzyMed/internal/google"
	"fuzzyMed/internal/rxnorm"
)


// Result captures the outcome for a single medication lookup.
type Result struct {
	OriginalName     string
	CorrectedName    string
	MatchedRxCUI     string
	MatchedName      string
	Score            string
	TTY              string
	TTYDescription   string
	IngredientRxCUI  string
	IngredientName   string
	HasMatch         bool
	WasTypoCorrected bool
	Error            string
}

// Maps summarizes derived data used by downstream consumers.
type Maps struct {
	NameToRxCUI       map[string]string
	BrandToIngredient map[string]string
	TypoToCorrect     map[string]string
	NoMatch           []string
}

// Logger is the logging function signature used by Processor.
type Logger func(format string, args ...any)

// Processor orchestrates cache usage, API calls, and result assembly.
type Processor struct {
	typoCache   *cache.TypoCache
	rxnormCache *cache.RxNormCache
	rxClient    *rxnorm.Client
	google      *google.Client
	delay       time.Duration
	logf        Logger
}
