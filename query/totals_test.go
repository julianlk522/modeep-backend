package query

import (
	"testing"

	"github.com/julianlk522/modeep/model"
)

func TestNewTotals(t *testing.T) {
	var totals model.Totals

	totals_sql := NewTotals()
	if err := TestClient.
		QueryRow(totals_sql.Text, totals_sql.Args...).
		Scan(&totals.Links,
			&totals.Clicks,
			&totals.Contributors,
			&totals.Likes,
			&totals.Tags,
			&totals.Summaries,
		); err != nil {
		t.Fatal(err)
	}

	// Verify "Auto Summary" not counted as a contributor
	// (9 total contributors in test data, 8 without auto summary)
	if totals.Contributors != 8 {
		t.Errorf("Expected 8 contributors, got %d", totals.Contributors)
	}
}
