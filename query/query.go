package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/fitm/error"
)

type Query struct {
	Text  string
	Args  []any
	Error error
}

func GetPeriodClause(period string) (clause string, err error) {
	var days int
	switch period {
	case "day":
		days = 1
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 365
	default:
		return "", e.ErrInvalidPeriod
	}

	return fmt.Sprintf("submit_date >= date('now', '-%d days')", days), nil
}

func GetCatsOptionalPluralOrSingularForms(cats []string) []string {
	modified_cats := make([]string, len(cats))
	for i := range cats {
		modified_cats[i] = WithOptionalPluralOrSingularForm(cats[i])
	}

	return modified_cats
}

func WithOptionalPluralOrSingularForm(cat string) string {
	lc_cat := strings.ToLower(cat)
	if strings.HasSuffix(lc_cat, "ss") {
		return fmt.Sprintf("(%s OR %s)", 
			GetCatSurroundedInDoubleQuotes(lc_cat), 
			GetCatSurroundedInDoubleQuotes(lc_cat+"es"),
		)
	} else if strings.HasSuffix(lc_cat, "sses") {
		return fmt.Sprintf("(%s OR %s)", 
			GetCatSurroundedInDoubleQuotes(lc_cat), 
			GetCatSurroundedInDoubleQuotes(strings.TrimSuffix(lc_cat, "es")),
		)
	} else if strings.HasSuffix(lc_cat, "s") {
		return fmt.Sprintf("(%s OR %s OR %s)", 
			GetCatSurroundedInDoubleQuotes(lc_cat), 
			GetCatSurroundedInDoubleQuotes(lc_cat+"es"), 
			GetCatSurroundedInDoubleQuotes(strings.TrimSuffix(lc_cat, "s")),
		)
	} else {
		return fmt.Sprintf("(%s OR %s)", 
			GetCatSurroundedInDoubleQuotes(lc_cat), 
			GetCatSurroundedInDoubleQuotes(lc_cat+"s"),
		)
	}
}

func GetCatsSurroundedInDoubleQuotes(cats []string) []string {
	modified_cats := make([]string, len(cats))
	for i := range cats {
		modified_cats[i] = GetCatSurroundedInDoubleQuotes(cats[i])
	}

	return modified_cats
}

func GetCatSurroundedInDoubleQuotes(cat string) string {
	return fmt.Sprintf(`"%s"`, cat)
}
