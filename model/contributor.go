package model

type Contributor struct {
	LoginName      string
	LinksSubmitted int
}

// OPTIONS
type TopContributorsOptions struct {
	CatFiltersWithSpellingVariants []string
	NeuteredCatFilters             []string
	SummaryContains                string
	URLContains                    string
	URLLacks                       string
	Period                         Period
}
