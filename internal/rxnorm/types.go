package rxnorm

// Go imports are file specific, even though types.go and client.go share the same package 
// of rxnorm, we still need seperate import for the packages we need to use for each file
import (
    "net/http" 
)

////////////////////////////////////////
// Public Struct 
////////////////////////////////////////
type Client struct {
	baseURL    string
	httpClient *http.Client
	minScore   int
}
// Match describes the outcome of an approximate term lookup.
type Match struct {
	RxCUI string
	Name  string
	Score string
}
// Ingredient represents the ingredient tied to a brand name.
type Ingredient struct {
	RxCUI string
	Name  string
}



////////////////////////////////////////
// Public Struct for JSON Mapping
////////////////////////////////////////

type approximateMatchResponse struct {
	ApproximateGroup struct {
		Candidate []struct {
			RxCUI  string `json:"rxcui"`
			Name   string `json:"name"`
			Score  string `json:"score"`
			Rank   string `json:"rank"`
			Source string `json:"source"`
		} `json:"candidate"`
	} `json:"approximateGroup"`
}

type allPropertiesResponse struct {
	PropConceptGroup struct {
		PropConcept []struct {
			PropCategory string `json:"propCategory"`
			PropName     string `json:"propName"`
			PropValue    string `json:"propValue"`
		} `json:"propConcept"`
	} `json:"propConceptGroup"`
}

type relatedResponse struct {
	RelatedGroup struct {
		ConceptGroup []struct {
			TTY               string `json:"tty"`
			ConceptProperties []struct {
				RxCUI string `json:"rxcui"`
				Name  string `json:"name"`
				TTY   string `json:"tty"`
			} `json:"conceptProperties"`
		} `json:"conceptGroup"`
	} `json:"relatedGroup"`
}
