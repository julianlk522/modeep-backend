package query

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/julianlk522/modeep/model"
)

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

	// Verify columns
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

	// Verify link_id
	tags_sql = NewTagRankings(test_link_id)

	tags_sql.Text = strings.Replace(tags_sql.Text,
		TAG_RANKINGS_BASE_FIELDS,
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

	// Verify columns
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

func TestNewTopGlobalCatCounts(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts()
	// No opportunity for counts_sql.Error to have been set

	rows, err := TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var counts []model.CatCount
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	// Verify capitalization variants in same tag are not double counted
	//  cat "infosec" appears in 2 tags, 1 of which also includes "Infosec"
	// so count should be 2 not 3

	for _, c := range counts {
		if c.Category == "infosec" && c.Count != 2 {
			t.Fatalf("Capitalization variants Infosec and infosec in same tag were double counted")
		}
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

	var counts []model.CatCount
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	// Verify counts
	for _, c := range counts {
		var count int32
		if err := TestClient.QueryRow(`SELECT count(id) as count 
		FROM LINKS 
		WHERE ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND ',' || global_cats || ',' LIKE '%' || ? || '%'`,
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

func TestNewTopGlobalCatCountsWithSummaryContaining(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().WithGlobalSummaryContaining("test")
	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}

	rows, err := TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var counts []model.CatCount
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	if len(counts) == 0 {
		t.Fatal("no top global cats returned")
	}

	// verify counts
	for _, c := range counts {
		var count int32
		if err := TestClient.QueryRow(`SELECT count(DISTINCT id) as count 
		FROM LINKS 
		WHERE ',' || global_cats || ',' LIKE '%,' || ? || ',%'
		AND global_summary LIKE '%' || ? || '%'`,
			c.Category,
			"test",
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

	// verify does not conflict w/ other methods
	counts_sql = NewTopGlobalCatCounts().
		SubcatsOfCats("flowers").
		WithGlobalSummaryContaining("test").
		WithURLContaining("www").
		WithURLLacking("donut").
		More()
	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}
	 
	rows, err = TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	counts = []model.CatCount{}
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	if len(counts) == 0 {
		t.Fatal("no top global cats returned")
	}

	for _, c := range counts {
		var count int32
		if err := TestClient.QueryRow(`SELECT count(DISTINCT id) as count 
		FROM LINKS 
		WHERE ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND global_summary LIKE '%' || ? || '%'
		AND url LIKE '%' || ? || '%'
		AND url NOT LIKE '%' || ? || '%'`,
			c.Category,
			"test",
			"www",
			"donut",
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

func TestNewTopGlobalCatCountsWithURLContaining(t *testing.T) {
	counts := []model.CatCount{}

	counts_sql := NewTopGlobalCatCounts().
		SubcatsOfCats(strings.Join(test_cats, ",")).
		WithURLContaining("GooGlE").
		More()
	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}

	rows, err := TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	for _, c := range counts {
		var count int32
		if err := TestClient.QueryRow(`SELECT count(id) as count 
		FROM LINKS 
		WHERE ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND url LIKE '%' || ? || '%'`,
			c.Category,
			"google",
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

func TestNewTopGlobalCatCountsWithURLLacking(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().
		SubcatsOfCats(strings.Join(test_cats, ",")).
		WithURLLacking("GooGlE")
	
	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}

	rows, err := TestClient.Query(counts_sql.Text, counts_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	counts := []model.CatCount{}
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	for _, c := range counts {
		var count int32
		if err := TestClient.QueryRow(`SELECT count(DISTINCT id) as count 
		FROM LINKS 
		WHERE ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND ',' || global_cats || ',' LIKE '%' || ? || '%'
		AND global_cats NOT IN (?, ?)
		AND url NOT LIKE '%' || ? || '%'
		`,
			c.Category,
			test_cats[0],
			test_cats[1],
			test_cats[0],
			test_cats[1],
			"google",
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
		{"all", true},
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

	// Verify no conflict with .SubcatsOfCats()
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
		"test":       21,
		"testing":    2,
		"tech":       2,
		"technology": 1,
	}

	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET)
	// No chance for matches_sql.Error to have been set

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
		"tech":       1,
		"technology": 1,
		"testing":    2,
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
