package query

import (
	"strings"
	"testing"

	"github.com/julianlk522/modeep/model"
)

func TestNewSummariesForLink(t *testing.T) {
	var test_link_id = "1"
	summaries_sql := NewSummariesForLink(test_link_id)
	rows, err := summaries_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}

	if len(cols) == 0 {
		t.Fatal("no columns")
	} else if len(cols) != 6 {
		t.Fatalf("wrong column count (got %d, want 6)", len(cols))
	}

	var test_cols = []struct {
		Want string
	}{
		{"sumid"},
		{"text"},
		{"ln"},
		{"last_updated"},
		{"like_count"},
		{"earliest_likers"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}

	// Verify no more than {SUMMARY_PAGE_LIMIT} rows
	var count int
	for rows.Next() {
		count++
	}
	if count > SUMMARIES_PAGE_LIMIT {
		t.Fatalf("got %d, want <= %d", count, SUMMARIES_PAGE_LIMIT)
	}

	// Verify link_ids
	summaries_sql.Text = strings.Replace(summaries_sql.Text, SUMMARIES_BASE_FIELDS, "SELECT sumid", 1)
	rows, err = summaries_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	summary_ids := []string{}
	for rows.Next() {
		var sumid string
		if err := rows.Scan(&sumid); err != nil {
			t.Fatal(err)
		}
		summary_ids = append(summary_ids, sumid)
	}

	for _, sumid := range summary_ids {
		var link_id string
		err = TestClient.QueryRow("SELECT link_id FROM Summaries WHERE id = ?", sumid).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		} else if link_id != test_link_id {
			t.Fatalf("got %s, want %s", link_id, test_link_id)
		}
	}
}

func TestSummariesAsSignedInUser(t *testing.T) {
	var test_link_id, test_user_id = "1", "2"
	summaries_sql := NewSummariesForLink(test_link_id).AsSignedInUser(test_user_id)
	rows, err := summaries_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}

	if len(cols) == 0 {
		t.Fatal("no columns")
	} else if len(cols) != 7 {
		t.Fatalf("wrong column count (got %d, want 6)", len(cols))
	}

	if rows.Next() {
		var s model.SummarySignedIn

		if err := rows.Scan(
			&s.ID,
			&s.Text,
			&s.SubmittedBy,
			&s.LastUpdated,
			&s.LikeCount,
			&s.EarliestLikers,
			&s.IsLiked,
		); err != nil {
			t.Fatal(err)
		}
	}
}
