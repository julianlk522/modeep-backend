package query

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"
)

// QUERY
type Query struct {
	Text  string
	Args  []any
	Error error
}

func (q *Query) ValidateAndExecuteRows() (*sql.Rows, error) {
	q.validateArgCount()
	if q.Error != nil {
		return nil, q.Error
	}

	rows, err := db.Client.Query(q.Text, q.Args...)
	return rows, err
}

func (q *Query) ValidateAndExecuteRow() (*sql.Row, error) {
	q.validateArgCount()
	if q.Error != nil {
		return nil, q.Error
	}

	row := db.Client.QueryRow(q.Text, q.Args...)
	return row, nil
}

func (q *Query) validateArgCount() {
	placeholders_in_text := strings.Count(q.Text, "?")
	arg_count := len(q.Args)
	if arg_count != placeholders_in_text {
		q.Error = e.ErrArgCountDoesNotMatchTextPlaceholders(arg_count, placeholders_in_text)
		log.Printf("Placeholders in SQL text: %d, Args bundled: %d", placeholders_in_text, arg_count)
		log.Printf("SQL text: %s", q.Text)
		log.Printf("Args: %v", q.Args)
	}
}

// CATS SPELLING VARIATIONS
// for FTS5 MATCH clause
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

// PERIOD
func getPeriodClause(period model.Period) (clause string, err error) {
	days, ok := model.ValidPeriodsInDays[period]
	if !ok {
		return "", e.ErrInvalidPeriod
	}
	return fmt.Sprintf("submit_date >= date('now', '-%d days')", days), nil
}