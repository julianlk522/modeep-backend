package query

import (
	"testing"

	"github.com/julianlk522/fitm/model"
)

func TestNewTotals(t *testing.T) {
	var totals model.Totals

	totals_sql := NewTotals()
	if err := TestClient.
		QueryRow(totals_sql.Text).
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
	// (8 total contributors in test data, 7 without auto summary)
	if totals.Contributors != 7 {
		t.Errorf("Expected 7 contributors, got %d", totals.Contributors)
	}
}