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
}

func TestNewGlobalCatsForLink(t *testing.T) {
	cats_sql := NewGlobalCatsForLink("1")
	if cats_sql.Error != nil {
		t.Fatal(cats_sql.Error)
	}
	
	var global_cats_str string
	if err := TestClient.QueryRow(cats_sql.Text, cats_sql.Args...).Scan(&global_cats_str); err != nil {
		if err == sql.ErrNoRows {
			t.Fatal("no global cats returned for link 1")
		} else {
			t.Fatal(err)
		}
	}

	// verify no capitalization or pluralization variants
	global_cats := strings.Split(global_cats_str, ",")
	found_cats := []string{}
	for _, cat := range global_cats {
		for _, found_cat := range found_cats {
			cat = strings.ToLower(cat)

			if cat == found_cat {
				t.Fatalf("found cat %s twice", cat)	
			}
			
			if cat + "s" == found_cat || 
			found_cat + "s" == cat || 
			cat + "es" == found_cat || 
			found_cat + "es" == cat {
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
	counts_sql := NewTopGlobalCatCounts().subcatsOfCats(strings.Join(test_cats, ","))
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
	counts_sql := NewTopGlobalCatCounts().withGlobalSummaryContaining("test")
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
		subcatsOfCats("flowers").
		withGlobalSummaryContaining("test").
		withURLContaining("www").
		withURLLacking("donut").
		more()
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
		subcatsOfCats(strings.Join(test_cats, ",")).
		withURLContaining("GooGlE").
		more()
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
		subcatsOfCats(strings.Join(test_cats, ",")).
		withURLLacking("GooGlE")
	
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
		tags_sql := NewTopGlobalCatCounts().duringPeriod(tp.Period)
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
			subcatsOfCats(strings.Join(test_cats, ",")).
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
	for _, sql := range []struct{ Text string; Args []any }{
	    {submitted_sql.Text, submitted_sql.Args},
	    {starred_sql.Text, starred_sql.Args},
	    {tagged_sql.Text, tagged_sql.Args},
	}{
		rows, err := TestClient.Query(sql.Text, sql.Args...)
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

func TestSpellfixMatchesWhileSubmittingLink(t *testing.T) {
	matches_sql := NewSpellfixMatchesForSnippet(TEST_SNIPPET).fromCatFiltersWhileSubmittingLink([]string{"flower"})
	matches_sql.ValidateAndExecuteRows()
	if matches_sql.Error != nil {
		t.Fatal(matches_sql.Error)
	}
}
