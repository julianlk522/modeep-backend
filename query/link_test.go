package query

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/julianlk522/modeep/model"
	mutil "github.com/julianlk522/modeep/model/util"
)

func TestNewTopLinks(t *testing.T) {
	links_sql := NewTopLinks()
	if links_sql.Error != nil {
		t.Fatal(links_sql.Error)
	}

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
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
	}

	var test_cols = []struct {
		Want string
	}{
		{"id"},
		{"url"},
		{"sb"},
		{"sd"},
		{"cats"},
		{"summary"},
		{"summary_count"},
		{"times_starred"},
		{"avg_stars"},
		{"earliest_starrers"},
		{"click_count"},
		{"tag_count"},
		{"img_file"},
		{"pages"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}
}

func TestFromCats(t *testing.T) {
	var test_cats = []struct {
		Cats  []string
		Valid bool
	}{
		{[]string{}, false},
		{[]string{""}, false},
		{[]string{"umvc3"}, true},
		{[]string{"umvc3", "flowers"}, true},
		{[]string{"YouTube", "c. viper"}, true},
	}

	for _, tc := range test_cats {
		// Cats only
		links_sql := NewTopLinks().fromCats(tc.Cats)
		if tc.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tc.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for cats %s", tc.Cats)
		}

		rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		defer rows.Close()

		// With period
		links_sql = links_sql.duringPeriod("month", "times_starred")
		if tc.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tc.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for cats %s", tc.Cats)
		}

		// If any cats provided, args should be cat_match and limit
		// in that order
		if len(tc.Cats) == 0 || len(tc.Cats) == 1 && tc.Cats[0] == "" {
			continue
		}

		if 
		links_sql.Args[len(links_sql.Args)-2] != strings.Join(tc.Cats, " ") && links_sql.Args[len(links_sql.Args)-1] != LINKS_PAGE_LIMIT {
			t.Fatalf("got %v, want %v (should be cat_match and limit in that order)", links_sql.Args, tc.Cats)
		}

		rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		defer rows.Close()
	}
}

func TestLinksWithGlobalSummaryContaining(t *testing.T) {
	// case-insensitive
	links_sql := NewTopLinks().withGlobalSummaryContaining("GoOgLe", "")

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.Link
	var pages int

	for rows.Next() {
		l := model.Link{}
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	if len(links) == 0 {
		t.Fatal("no links")
	}

	for _, l := range links {
		if !strings.Contains(strings.ToLower(l.Summary), "google") {
			t.Fatalf("got %s, want %s", l.Summary, "google")
		}
	}

	// no conflct w/ other methods
	links_sql = NewTopLinks().
		fromCats([]string{"test"}).
		withGlobalSummaryContaining("GoOgLe", "").
		withURLContaining("www", "").
		withURLLacking("something", "").
		AsSignedInUser(TEST_USER_ID).
		nsfw().
		sortBy("newest").
		Page(1).
		duringPeriod("all", "newest")
	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	lsi := []model.LinkSignedIn{}
	for rows.Next() {
		l := model.LinkSignedIn{}
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}

		lsi = append(lsi, l)
	}

	if len(lsi) == 0 {
		t.Fatal("no links")
	}

	for _, l := range lsi {
		if !strings.Contains(strings.ToLower(l.Summary), "google") {
			t.Fatalf("got %s, want %s", l.Summary, "google")
		}
	}
}

func TestLinksWithURLContaining(t *testing.T) {
	links_sql := NewTopLinks().withURLContaining("GoOgLe", "")

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.Link
	var pages int

	for rows.Next() {
		l := model.Link{}
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	if len(links) == 0 {
		t.Fatal("no links")
	}

	for _, l := range links {
		if !strings.Contains(l.URL, "google") {
			t.Fatalf("got %s, want containing google", l.URL)
		}
	}

	// combined with other methods
	links_sql = NewTopLinks().
		fromCats([]string{"flowers"}).
		withURLContaining("google", "times_starred").
		AsSignedInUser(TEST_USER_ID).
		sortBy("newest")
	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	links_signed_in := []model.LinkSignedIn{}

	for rows.Next() {
		l := model.LinkSignedIn{}
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}

		links_signed_in = append(links_signed_in, l)
	}

	if len(links_signed_in) == 0 {
		t.Fatal("no links")
	}

	for _, l := range links_signed_in {
		if !strings.Contains(l.URL, "google") {
			t.Fatalf("got %s, want containing google", l.URL)
		}
	}
}

func TestLinksWithURLLacking(t *testing.T) {
	// case-insensitive
	links_sql := NewTopLinks().withURLLacking("gOoGlE", "")

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.Link
	var pages int

	for rows.Next() {
		l := model.Link{}
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	if len(links) == 0 {
		t.Fatal("no links")
	}

	for _, l := range links {
		if strings.Contains(l.URL, "google") {
			t.Fatalf("got %s, should not contain google", l.URL)
		}
	}

	// combined with other methods
	links_sql = NewTopLinks().
		fromCats([]string{"umvc3"}).
		withURLLacking("gOOgle", "times_starred").
		AsSignedInUser(TEST_USER_ID).
		sortBy("newest")
	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	var links_signed_in []model.LinkSignedIn

	for rows.Next() {
		l := model.LinkSignedIn{}
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}

		links_signed_in = append(links_signed_in, l)
	}

	if len(links_signed_in) == 0 {
		t.Fatal("no links")
	}

	for _, l := range links_signed_in {
		if strings.Contains(l.URL, "google") {
			t.Fatalf("got %s, should not contain google", l.URL)
		}
	}
}

func TestLinksDuringPeriod(t *testing.T) {
	var test_periods = []struct {
		Period string
		Valid  bool
	}{
		{"day", true},
		{"week", true},
		{"month", true},
		{"year", true},
		{"all", true},
		{"gobblety gook", false},
	}

	for _, tp := range test_periods {
		// Period only
		links_sql := NewTopLinks().duringPeriod(tp.Period, "")
		if tp.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tp.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		rows.Close()

		// With cats
		links_sql = links_sql.fromCats([]string{"umvc3"})
		if tp.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tp.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		rows.Close()
	}
}

func TestLinksSortBy(t *testing.T) {
	var test_sorts = []struct {
		Sort  string
		Valid bool
	}{
		{"newest", true},
		{"times_starred", true},
		{"avg_stars", true},
		{"oldest", true},
		{"clicks", true},
		{"random", false},
		{"invalid", false},
	}

	var pages int

	for _, ts := range test_sorts {
		links_sql := NewTopLinks().sortBy(ts.Sort)
		if ts.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !ts.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for sort %s", ts.Sort)
		}

		rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()

		// Scan
		var links []model.Link
		for rows.Next() {
			l := model.Link{}
			err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.Summary,
				&l.SummaryCount,
				&l.TimesStarred,
				&l.AvgStars,
				&l.EarliestStarrers,
				&l.ClickCount,
				&l.TagCount,
				&l.PreviewImgFilename,
				&pages,
			)
			if err != nil {
				t.Fatal(err)
			}
			links = append(links, l)
		}

		if !ts.Valid {
			continue
		}

		// Verify results correctly sorted
		switch ts.Sort {
			case "times_starred":
				var last_star_count int64 = 999 // arbitrary high number
				for _, link := range links {
					if link.TimesStarred > last_star_count {
						t.Fatalf("link like count %d above previous min %d", link.TimesStarred, last_star_count)
					} else if link.TimesStarred < last_star_count {
						last_star_count = link.TimesStarred
					}
				}
			case "avg_stars":
				var last_avg_stars float32 = 999
				for _, link := range links {
					if link.AvgStars > last_avg_stars {
						t.Fatalf("link avg stars %f above previous min %f", link.AvgStars, last_avg_stars)
					} else if link.AvgStars < last_avg_stars {
						last_avg_stars = link.AvgStars
					}
				}
			case "newest":
				last_date := time.Now() // most recent
				for _, link := range links {
					sd, err := time.Parse("2006-01-02T15:04:05Z07:00", link.SubmitDate)
					if err != nil {
						t.Fatal(err)
					}

					if sd.After(last_date) {
						t.Fatalf("link date %s after last date %s", sd, last_date)
					} else if sd.Before(last_date) {
						last_date = sd
					}
				}
			case "clicks":
				var last_click_count int64 = 999 // arbitrary high number
				for _, link := range links {
					if link.ClickCount > last_click_count {
						t.Fatalf("link click count %d above previous min %d", link.ClickCount, last_click_count)
					} else if link.ClickCount < last_click_count {
						last_click_count = link.ClickCount
					}
				}
		}

	}
}

func TestAsSignedInUser(t *testing.T) {
	links_sql := NewTopLinks().AsSignedInUser(TEST_USER_ID)
	if links_sql.Error != nil {
		t.Fatal(links_sql.Error)
	}

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
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
	}

	var test_cols = []struct {
		Want string
	}{
		{"id"},
		{"url"},
		{"sb"},
		{"sd"},
		{"cats"},
		{"summary"},
		{"summary_count"},
		{"times_starred"},
		{"avg_stars"},
		{"earliest_starrers"},
		{"click_count"},
		{"tag_count"},
		{"img_file"},
		{"pages"},
		{"stars_assigned"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}

	var expected_args = []any{
		mutil.EARLIEST_STARRERS_LIMIT, 
		TEST_USER_ID, 
		LINKS_PAGE_LIMIT,
	}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("arg %d: got %v, want %v", i, arg, expected_args[i])
		}
	}

	// Verify no conflict with .FromCats()
	links_sql = NewTopLinks().fromCats(test_cats).AsSignedInUser(TEST_USER_ID)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}

	// "go AND coding" modified to include plural/singular variations
	expected_args = []any{
		mutil.EARLIEST_STARRERS_LIMIT, 
		TEST_USER_ID, 
		WithOptionalPluralOrSingularForm("go") + " AND " + WithOptionalPluralOrSingularForm("coding"), 
		LINKS_PAGE_LIMIT,
	}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("arg %d: got %v, want %v", i, arg, expected_args[i])
		}
	}
}

func TestNSFW(t *testing.T) {
	links_sql := NewTopLinks().nsfw()
	// No opportunity for links_sql.Error to have been set

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Verify no conflict with other filter methods
	links_sql = NewTopLinks().
		fromCats([]string{"search", "engine", "NSFW"}).
		duringPeriod("year", "times_starred").
		AsSignedInUser(TEST_USER_ID).
		sortBy("newest").
		Page(1).
		nsfw()

	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	id_of_test_link_having_nsfw_cats := "76"
	var l model.LinkSignedIn
	var pages int
	// there is 
	for rows.Next() {
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		} else if l.ID != id_of_test_link_having_nsfw_cats {
			t.Fatalf("got %s, want %s", l.ID, id_of_test_link_having_nsfw_cats)
		}
	}

	// Verify link not present using same query without .NSFW()
	links_sql = NewTopLinks().
		fromCats([]string{"search", "engine", "NSFW"}).
		duringPeriod("year", "newest").
		AsSignedInUser(TEST_USER_ID).
		sortBy("oldest").
		Page(1)

	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TimesStarred,
			&l.AvgStars,
			&l.EarliestStarrers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		} else if l.ID == id_of_test_link_having_nsfw_cats {
			t.Fatalf("got %s, want nil", l.ID)
		}
	}
}

func TestPage(t *testing.T) {
	var test_cases = []struct {
		Page         int
		WantLimitArg int
	}{
		{0, LINKS_PAGE_LIMIT},
		{1, LINKS_PAGE_LIMIT + 1},
		{2, LINKS_PAGE_LIMIT + 1},
		{3, LINKS_PAGE_LIMIT + 1},
	}

	var links_sql = NewTopLinks()

	for _, tc := range test_cases {
		links_sql = links_sql.Page(tc.Page)
		if links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		}

		if tc.Page > 1 {
			limit_arg := links_sql.Args[len(links_sql.Args)-2]
			offset_arg := links_sql.Args[len(links_sql.Args)-1]

			if limit_arg != tc.WantLimitArg {
				t.Fatalf("got %d, want %d", limit_arg, tc.WantLimitArg)
			} else if offset_arg != (tc.Page-1)*LINKS_PAGE_LIMIT {
				t.Fatalf("got %d, want %d", offset_arg, tc.WantLimitArg)
			}

			continue
		}

		if links_sql.Args[len(links_sql.Args)-1] != tc.WantLimitArg {
			t.Fatalf("got %d, want %d", links_sql.Args[len(links_sql.Args)-1], tc.WantLimitArg)
		}
	}

	// Verify no conflict with other methods
	links_sql = NewTopLinks().
		fromCats(test_cats).
		duringPeriod("year", "times_starred").
		sortBy("newest").
		AsSignedInUser(TEST_USER_ID).
		nsfw().
		Page(2)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}

	// "go AND coding" modified to include plural/singular variations
	var expected_args = []any{
		mutil.EARLIEST_STARRERS_LIMIT, 
		TEST_USER_ID, 
		WithOptionalPluralOrSingularForm("go") + " AND " + WithOptionalPluralOrSingularForm("coding"), 
		LINKS_PAGE_LIMIT + 1, 
		LINKS_PAGE_LIMIT,
	}

	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("got %v, want %v", arg, expected_args[i])
		}
	}
}

func TestCountNSFWLinks(t *testing.T) {
	// without NSFW params enabled
	links_sql := NewTopLinks().CountNSFWLinks(false)
	if links_sql.Error != nil {
		t.Fatal(links_sql.Error)
	}

	var nsfw_links int
	if err := TestClient.QueryRow(links_sql.Text, links_sql.Args...).Scan(&nsfw_links); err != nil {
		t.Fatal(err)
	}
	
	// with NSFW params
	links_sql = NewTopLinks().
		nsfw().
		// NOTE: this breaks the query if passed "true" and .NSFW() is not called first
		// (which shouldn't happen)
		CountNSFWLinks(true)
	if links_sql.Error != nil {
		t.Fatal(links_sql.Error)
	}

	if err := TestClient.QueryRow(links_sql.Text, links_sql.Args...).Scan(&nsfw_links); err != nil {
		t.Fatalf(
			"err: %v, sql text was %s, args were %v",
			err,
			links_sql.Text,
			links_sql.Args,
		)
	}

	// combined with other methods
	links_sql = NewTopLinks().
		fromCats(test_cats).
		duringPeriod("year", "times_starred").
		sortBy("newest").
		AsSignedInUser(TEST_USER_ID).
		CountNSFWLinks(false)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatalf(
			"err: %v, sql text was %s, args were %v",
			err,
			links_sql.Text,
			links_sql.Args,
		)
	}

	// MORE COMBINATIONSSSSS
	links_sql = NewTopLinks().
		fromCats(test_cats).
		duringPeriod("all", "average_stars").
		sortBy("average_stars").
		withURLContaining("www", "average_stars").
		withURLLacking(".com", "average_stars").
		withGlobalSummaryContaining("test", "average_stars").
		AsSignedInUser(TEST_USER_ID).
		nsfw().
		CountNSFWLinks(true)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}
}
