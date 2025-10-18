package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/modeep/error"
)

type Query struct {
	Text  string
	Args  []any
	Error error
}

func getPeriodClause(period string) (clause string, err error) {
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
		modified_cats[i] = withOptionalPluralOrSingularForm(cats[i])
	}

	return modified_cats
}

func withOptionalPluralOrSingularForm(cat string) string {
	lc_cat := strings.ToLower(cat)
	if strings.HasSuffix(lc_cat, "ss") {
		return fmt.Sprintf("(%s OR %s)", 
			getCatSurroundedInDoubleQuotes(lc_cat), 
			getCatSurroundedInDoubleQuotes(lc_cat+"es"),
		)
	} else if strings.HasSuffix(lc_cat, "sses") {
		return fmt.Sprintf("(%s OR %s)", 
			getCatSurroundedInDoubleQuotes(lc_cat), 
			getCatSurroundedInDoubleQuotes(strings.TrimSuffix(lc_cat, "es")),
		)
	} else if strings.HasSuffix(lc_cat, "s") {
		return fmt.Sprintf("(%s OR %s OR %s)", 
			getCatSurroundedInDoubleQuotes(lc_cat), 
			getCatSurroundedInDoubleQuotes(lc_cat+"es"), 
			getCatSurroundedInDoubleQuotes(strings.TrimSuffix(lc_cat, "s")),
		)
	} else {
		return fmt.Sprintf("(%s OR %s)", 
			getCatSurroundedInDoubleQuotes(lc_cat), 
			getCatSurroundedInDoubleQuotes(lc_cat+"s"),
		)
	}
}

func getCatSurroundedInDoubleQuotes(cat string) string {
	return fmt.Sprintf(`"%s"`, cat)
}
