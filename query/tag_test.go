package query

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/julianlk522/fitm/model"
)

// Tags Page Link
func TestNewTagPageLink(t *testing.T) {
	test_link_id := "1"

	// signed out
	tag_sql := NewTagPageLink(test_link_id)
	if tag_sql.Error != nil {
		t.Fatal(tag_sql.Error)
	}

	var l model.Link
	if err := TestClient.QueryRow(tag_sql.Text, tag_sql.Args...).Scan(
		&l.ID,
		&l.URL,
		&l.SubmittedBy,
		&l.SubmitDate,
		&l.Cats,
		&l.Summary,
		&l.SummaryCount,
		&l.LikeCount,
		&l.ImgURL,
	); err != nil {
		t.Fatal(err)
	}

	if l.ID != test_link_id {
		t.Fatalf("got %s, want %s", l.ID, test_link_id)
	}

	// signed in
	tag_sql = tag_sql.AsSignedInUser(test_req_user_id)
	if tag_sql.Error != nil {
		t.Fatal(tag_sql.Error)
	}

	var lsi model.LinkSignedIn
	if err := TestClient.QueryRow(tag_sql.Text, tag_sql.Args...).Scan(
		&lsi.ID,
		&lsi.URL,
		&lsi.SubmittedBy,
		&lsi.SubmitDate,
		&lsi.Cats,
		&lsi.Summary,
		&lsi.TagCount,
		&lsi.LikeCount,
		&lsi.ImgURL,
		&lsi.IsLiked,
		&lsi.IsCopied,
	); err != nil {
		t.Fatal(err)
	}
}

// Tag Rankings (cat overlap scores)
func TestNewTagRankings(t *testing.T) {
	test_link_id := "1"

	tags_sql := NewTagRankings(test_link_id)
	if tags_sql.Error != nil {
		t.Fatal(tags_sql.Error)
	}

	rows, err := TestClient.Query(tags_sql.Text, tags_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// verify first row columns only (rest are same)
	if rows.Next() {
		var tr model.TagRanking
		if err := rows.Scan(
			&tr.LifeSpanOverlap,
			&tr.Cats,
		); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatalf("no overlap scores for test link %s", test_link_id)
	}

	// verify correct link_id (test _FromLink())
	// reset and modify fields
	tags_sql = NewTagRankings(test_link_id)

	tags_sql.Text = strings.Replace(tags_sql.Text,
		TOP_OVERLAP_SCORES_BASE_FIELDS,
		`SELECT link_id`,
		1)
	tags_sql.Text = strings.Replace(tags_sql.Text,
		"ORDER BY lifespan_overlap DESC",
		"",
		1)

	rows, err = TestClient.Query(tags_sql.Text, tags_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if rows.Next() {
		var link_id string
		if err := rows.Scan(&link_id); err != nil {
			t.Fatal(err)
		}

		if link_id != test_link_id {
			t.Fatalf("got %s, want %s", link_id, test_link_id)
		}
	} else {
		t.Fatalf("failed link_id check with modified query: test link %s NOT returned", test_link_id)
	}

	// Public rankings
	tags_sql = NewTagRankings(test_link_id).Public()
	if tags_sql.Error != nil {
		t.Fatal(tags_sql.Error)
	}

	rows, err = TestClient.Query(tags_sql.Text, tags_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// verify columns
	if rows.Next() {
		var tr model.TagRankingPublic

		if err := rows.Scan(
			&tr.LifeSpanOverlap,
			&tr.Cats,
			&tr.SubmittedBy,
			&tr.LastUpdated,
		); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatalf("no public tag rankings for test link %s", test_link_id)
	}
}

// All Global Cats
func TestNewTopGlobalCatCounts(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts()
	// no opportunity for counts_sql.Error to have been set so no need to check

	_, err := TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewTopGlobalCatCountsSubcatsOfCats(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().SubcatsOfCats(strings.Join(test_cats, ","))
	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}

	rows, err := TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	// scan
	var counts []model.CatCount
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	// verify counts
	for _, c := range counts {
		var count int32
		if err := TestClient.QueryRow( `SELECT count(id) as count 
		FROM LINKS 
		WHERE global_cats LIKE '%' || ? || '%'
		AND global_cats LIKE '%' || ? || '%'
		AND global_cats LIKE '%' || ? || '%'`,
		test_cats[0],
		test_cats[1],
		c.Category,
	).Scan(&count); err != nil {
			t.Fatal(err)
		} else if count != c.Count {
			t.Fatalf(
				"got %d, want %d for cat %s",
				count,
				c.Count,
				c.Category,
			)
		}
	}
}

func TestNewTopGlobalCatCountsDuringPeriod(t *testing.T) {
	var test_periods = []struct {
		Period string
		Valid  bool
	}{
		{"day", true},
		{"week", true},
		{"month", true},
		{"year", true},
		{"all", false},
		{"invalid_period", false},
	}

	for _, tp := range test_periods {
		tags_sql := NewTopGlobalCatCounts().DuringPeriod(tp.Period)
		if tp.Valid && tags_sql.Error != nil {
			t.Fatalf("unexpected error for period %s", tp.Period)
		} else if !tp.Valid && tags_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		_, err := TestClient.Query(tags_sql.Text, tags_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
	}

	// verify no conflict with .SubcatsOfCats()
	for _, tp := range test_periods {
		tags_sql := NewTopGlobalCatCounts().
			SubcatsOfCats(strings.Join(test_cats, ",")).
			DuringPeriod(tp.Period)
		if tp.Valid && tags_sql.Error != nil {
			t.Fatalf("unexpected error for period %s", tp.Period)
		} else if !tp.Valid && tags_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}
	}
}

const TEST_SNIPPET = "test"

func TestNewSpellfixMatchesForSnippet(t *testing.T) {
	var expected_rankings = map[string]int{
		"test":       11,
		"tech":       2,
		"technology": 1,
	}

	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET)
	// no chance for sql.Error to have been set so no need to check

	rows, err := TestClient.Query(matches_sql.Text, matches_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var word string
		var rank int

		if err := rows.Scan(&word, &rank); err != nil {
			t.Fatal(err)
		} else if expected_rankings[word] != rank {
			t.Fatalf("got %d, want %d for word %s", rank, expected_rankings[word], word)
		}
	}
}

func TestOmitCats(t *testing.T) {
	var expected_rankings = map[string]int{
		// "test": 11, // filter out
		"tech":       2,
		"technology": 1,
	}
	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET)
	err := matches_sql.OmitCats([]string{TEST_SNIPPET})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := TestClient.Query(matches_sql.Text, matches_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var word string
		var count int
		if err := rows.Scan(&word, &count); err != nil {
			t.Fatal(err)
		} else if expected_rankings[word] != count {
			t.Fatalf("got %d, want %d for word %s", count, expected_rankings[word], word)
		}
	}
}
