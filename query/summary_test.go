package query

import (
	"strings"
	"testing"

	"github.com/julianlk522/fitm/model"
)

// Summaries Page Link
func TestNewSummaryPageLink(t *testing.T) {
	link_sql := NewSummaryPageLink("1")

	if link_sql.Error != nil {
		t.Fatal(link_sql.Error)
	}

	rows, err := TestClient.Query(link_sql.Text, link_sql.Args...)
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
	} else if len(cols) != 9 {
		t.Fatal("too few columns")
	}

	var test_cols = []struct {
		Want string
	}{
		{"link_id"},
		{"url"},
		{"sb"},
		{"sd"},
		{"cats"},
		{"summary"},
		{"like_count"},
		{"tag_count"},
		{"img_url"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}
}

func Test_SummaryPageLinkFromID(t *testing.T) {
	var test_link_id = "1"
	link_sql := NewSummaryPageLink(test_link_id)

	if link_sql.Error != nil {
		t.Fatal(link_sql.Error)
	}

	var l model.Link
	err := TestClient.QueryRow(link_sql.Text, link_sql.Args...).Scan(
		&l.ID,
		&l.URL,
		&l.SubmittedBy,
		&l.SubmitDate,
		&l.Cats,
		&l.Summary,
		&l.TagCount,
		&l.LikeCount,
		&l.ImgURL,
	)
	if err != nil {
		t.Fatal(err)
	}

	if l.ID != test_link_id {
		t.Fatalf("got %s, want %s", l.ID, test_link_id)
	}
}

func TestSummaryPageLinkAsSignedInUser(t *testing.T) {
	var test_link_id, test_user_id = "1", "2"
	link_sql := NewSummaryPageLink(test_link_id).AsSignedInUser(test_user_id)

	if link_sql.Error != nil {
		t.Fatal(link_sql.Error)
	}

	var l model.LinkSignedIn
	err := TestClient.QueryRow(link_sql.Text, link_sql.Args...).Scan(
		&l.ID,
		&l.URL,
		&l.SubmittedBy,
		&l.SubmitDate,
		&l.Cats,
		&l.Summary,
		&l.TagCount,
		&l.LikeCount,
		&l.ImgURL,
		&l.IsLiked,
		&l.IsCopied,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// Summaries for Link
func TestNewSummariesForLink(t *testing.T) {
	var test_link_id = "1"
	summaries_sql := NewSummariesForLink(test_link_id)
	if summaries_sql.Error != nil {
		t.Fatal(summaries_sql.Error)
	}

	rows, err := TestClient.Query(summaries_sql.Text, summaries_sql.Args...)
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
	} else if len(cols) != 5 {
		t.Fatalf("wrong column count (got %d, want 5)", len(cols))
	}

	var test_cols = []struct {
		Want string
	}{
		{"sumid"},
		{"text"},
		{"ln"},
		{"last_updated"},
		{"like_count"},
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

	rows, err = TestClient.Query(summaries_sql.Text, summaries_sql.Args...)
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

func TestNewSummariesAsSignedInUser(t *testing.T) {
	var test_link_id, test_user_id = "1", "2"
	summaries_sql := NewSummariesForLink(test_link_id).AsSignedInUser(test_user_id)

	if summaries_sql.Error != nil {
		t.Fatal(summaries_sql.Error)
	}

	rows, err := TestClient.Query(summaries_sql.Text, summaries_sql.Args...)
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

	// only necessary to test first row since they will all be the same
	if rows.Next() {
		var s model.SummarySignedIn

		if err := rows.Scan(
			&s.ID,
			&s.Text,
			&s.SubmittedBy,
			&s.LastUpdated,
			&s.LikeCount,
			&s.IsLiked,
		); err != nil {
			t.Fatal(err)
		}
	}
}
