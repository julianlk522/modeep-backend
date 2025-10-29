package query

import (
	"database/sql"
	"slices"
	"strings"
	"testing"

	"github.com/julianlk522/modeep/model"
)

func TestNewTagRankings(t *testing.T) {
	test_link_id := "1"
	tags_sql := NewTagRankingsForLink(test_link_id)
	rows, err := tags_sql.ValidateAndExecuteRows()
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
			&tr.SubmittedBy,
			&tr.LastUpdated,
		); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatalf("no results for link %s", test_link_id)
	}

	// Verify link_id
	tags_sql = NewTagRankingsForLink(test_link_id)

	tags_sql.Text = strings.Replace(tags_sql.Text,
		`SELECT
	(julianday('now') - julianday(t.last_updated)) / (julianday('now') - julianday(l.submit_date)) * 100 AS lifespan_overlap, 
	t.cats, 
	t.submitted_by, 
	t.last_updated`,
		`SELECT t.link_id`,
		1)
	tags_sql.Text = strings.Replace(tags_sql.Text,
		"ORDER BY lifespan_overlap DESC",
		"",
		1)

	rows, err = tags_sql.ValidateAndExecuteRows()
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
}

func TestNewGlobalCatsForLink(t *testing.T) {
	cats_sql := NewGlobalCatsForLink("1")
	row, err := cats_sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var global_cats_str string
	if err := row.Scan(&global_cats_str); err != nil {
		if err == sql.ErrNoRows {
			t.Fatal("no global cats returned for link 1")
		} else {
			t.Fatal(err)
		}
	}

	// Verify no capitalization or pluralization variants
	global_cats := strings.Split(global_cats_str, ",")
	found_cats := []string{}
	for _, cat := range global_cats {
		for _, found_cat := range found_cats {
			cat = strings.ToLower(cat)

			if cat == found_cat {
				t.Fatalf("found cat %s twice", cat)
			}

			if cat+"s" == found_cat ||
				found_cat+"s" == cat ||
				cat+"es" == found_cat ||
				found_cat+"es" == cat {
				t.Fatalf(
					"found cat %s and %s are singular or plural variations of each other",
					found_cat,
					cat,
				)
			}

			found_cats = append(found_cats, cat)
		}
	}
}

func TestNewTopGlobalCatCounts(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts()
	if _, err := counts_sql.ValidateAndExecuteRows(); err != nil {
		t.Fatal(err)
	}
}

func TestTopGlobalCatCountsFromCatFilters(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().fromCatFilters(test_cats)
	rows, err := counts_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	if !rows.Next() {
		t.Fatal("no rows")
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

	// None of the counts should be capitalization/pluralization variants of
	// cat filters
	var lc_cat_filters []string
	for _, cf := range test_cats {
		lc_cat_filters = append(lc_cat_filters, strings.ToLower(cf))
	}

	for _, c := range counts {
		c.Category = strings.ToLower(c.Category)

		for _, lc_cf := range lc_cat_filters {
			if c.Category+"s" == lc_cf ||
				lc_cf+"s" == c.Category ||
				c.Category+"es" == lc_cf ||
				lc_cf+"es" == c.Category {
				t.Fatalf(
					"found cat %s and %s resemble each other",
					lc_cf,
					c.Category,
				)
			}
		}
	}
}

func TestTopGlobalCatCountsFromNeuteredCatFilters(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().fromNeuteredCatFilters(test_cats)
	rows, err := counts_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatalf(
			"err: %v, sql text: %s, args: %v",
			err,
			counts_sql.Text,
			counts_sql.Args,
		)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no rows")
	}

	var counts []model.CatCount
	for rows.Next() {
		var c model.CatCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, c)
	}

	for _, cc := range counts {
		if slices.Contains(test_cats, cc.Category) {
			t.Fatalf(
				"cat %s was not neutered, sql text: %s, args: %v",
				cc.Category,
				counts_sql.Text,
				counts_sql.Args,
			)
		}
	}
}

func TestTopGlobalCatCountsWhereSummaryContains(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().whereGlobalSummaryContains("test")
	rows, err := counts_sql.ValidateAndExecuteRows()
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
		fromCatFilters([]string{"flowers"}).
		whereGlobalSummaryContains("test").
		whereURLContains("www").
		whereURLLacks("donut").
		more()
	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}

	rows, err = counts_sql.ValidateAndExecuteRows()
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

func TestTopGlobalCatCountsWhereURLContains(t *testing.T) {
	test_url_snippet := "gOoGlE"
	counts_sql := NewTopGlobalCatCounts().
		whereURLContains(test_url_snippet).
		more()
	rows, err := counts_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	if !rows.Next() {
		t.Fatalf(
			"no rows, sql text: %s, args: %v",
			counts_sql.Text,
			counts_sql.Args,
		)
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
		WHERE url LIKE '%' || ? || '%'
		AND id IN (
			SELECT link_id 
			FROM global_cats_fts 
			WHERE global_cats MATCH ?
		)`,
			test_url_snippet,
			withOptionalPluralOrSingularForm(c.Category),
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

func TestTopGlobalCatCountsWhereURLLacks(t *testing.T) {
	counts_sql := NewTopGlobalCatCounts().
		fromCatFilters(test_cats).
		whereURLLacks("GooGlE")

	if counts_sql.Error != nil {
		t.Fatal(counts_sql.Error)
	}

	rows, err := counts_sql.ValidateAndExecuteRows()
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

func TestTopGlobalCatCountsDuringPeriod(t *testing.T) {
	var test_periods = []struct {
		Period model.Period
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
		tags_sql := NewTopGlobalCatCounts().duringPeriod(tp.Period)
		if tp.Valid && tags_sql.Error != nil {
			t.Fatalf("unexpected error for period %s", tp.Period)
		} else if !tp.Valid && tags_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		if !tp.Valid {
			continue
		}

		_, err := tags_sql.ValidateAndExecuteRows()
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
	}

	// Verify no conflict with .fromCatFilters()
	for _, tp := range test_periods {
		tags_sql := NewTopGlobalCatCounts().
			fromCatFilters(test_cats).
			duringPeriod(tp.Period)
		if tp.Valid && tags_sql.Error != nil {
			t.Fatalf("unexpected error for period %s", tp.Period)
		} else if !tp.Valid && tags_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}
	}
}

const TEST_SNIPPET = "test"

func TestNewSpellfixMatchesForSnippet(t *testing.T) {
	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET)
	rows, err := matches_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var word string
		var rank int

		if err := rows.Scan(&word, &rank); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSpellfixMatchesFromTmap(t *testing.T) {
	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET).fromTmap(TEST_LOGIN_NAME)
	rows, err := matches_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	var found_cats []model.CatCount

	for rows.Next() {
		var word string
		var count int32
		if err := rows.Scan(&word, &count); err != nil {
			t.Fatal(err)
		}

		found_cats = append(found_cats, model.CatCount{
			Category: word,
			Count:    count,
		})
	}

	// verify all found cats are on test user's Treasure Map
	var submitted_sql = NewTmapSubmitted(TEST_LOGIN_NAME)
	var starred_sql = NewTmapStarred(TEST_LOGIN_NAME)
	var tagged_sql = NewTmapTagged(TEST_LOGIN_NAME)

	var all_tmap_links []model.TmapLink
	for _, q := range []*Query{
		{submitted_sql.Text, submitted_sql.Args, nil},
		{starred_sql.Text, starred_sql.Args, nil},
		{tagged_sql.Text, tagged_sql.Args, nil},
	} {
		rows, err := q.ValidateAndExecuteRows()
		if err != nil {
			t.Fatal(err)
		}

		for rows.Next() {
			var l model.TmapLink
			if err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.CatsFromUser,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.AvgStars,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
			); err != nil {
				t.Fatal(err)
			}

			all_tmap_links = append(all_tmap_links, l)
		}
	}

	var all_tmap_cats []string
	for _, l := range all_tmap_links {
		// doesn't really matter if this collects duplicates
		all_tmap_cats = append(all_tmap_cats, strings.Split(l.Cats, ",")...)
	}

	for _, cat := range found_cats {
		if !slices.Contains(all_tmap_cats, cat.Category) {
			t.Fatalf("cat %s not found on user's Tmap", cat.Category)
		}
	}
}

func TestSpellfixMatchesFromCatFilters(t *testing.T) {
	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET).fromCatFilters([]string{TEST_SNIPPET})
	rows, err := matches_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var word string
		var count int
		if err := rows.Scan(&word, &count); err != nil {
			t.Fatal(err)
		}
	}

	// TODO confirm pattern matching of top global cats
}

func TestSpellfixMatchesWhileAddingCats(t *testing.T) {
	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET).fromCatFiltersWhileAddingCats([]string{"flower"})
	matches_sql.ValidateAndExecuteRows()
	if matches_sql.Error != nil {
		t.Fatal(matches_sql.Error)
	}
}
