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
	rows, err := links_sql.ValidateAndExecuteRows()
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

func TestTopLinksFromCatFilters(t *testing.T) {
	var test_cats = []struct {
		CatFilters []string
		Valid      bool
	}{
		{[]string{}, false},
		{[]string{""}, false},
		{[]string{"umvc3"}, true},
		{[]string{"umvc3", "flowers"}, true},
		{[]string{"umvc3", "flowers", "test"}, true},
	}

	for _, tc := range test_cats {
		links_sql := NewTopLinks().fromCatFilters(tc.CatFilters)
		if tc.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tc.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for cats %s", tc.CatFilters)
		}

		if tc.Valid {
			if _, err := links_sql.ValidateAndExecuteRows(); err != nil {
				t.Fatalf(
					"got error %s, sql text was %s, args were %v",
					err,
					links_sql.Text,
					links_sql.Args,
				)
			}

			// With period
			links_sql = links_sql.duringPeriod("month")
			if _, err := links_sql.ValidateAndExecuteRows(); err != nil && err != sql.ErrNoRows {
				t.Fatal(err)
			}
		}
	}
}

func TestTopLinksFromNeuteredCatFilters(t *testing.T) {
	var test_neutered_cats = []string{"umvc3", "best", "game", "ever"}
	links_sql := NewTopLinks().fromNeuteredCatFilters(test_neutered_cats)
	rows, err := links_sql.ValidateAndExecuteRows()
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no rows")
	}

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

	// none of the links should have any member of test_neutered_cats
	// in their global cats
	for _, l := range links {
		for lc := range strings.SplitSeq(l.Cats, " ") {
			for _, nc := range test_neutered_cats {
				if nc == lc {
					t.Fatalf("link %s has global cat %s", l.URL, nc)
				}
			}
		}
	}

	// with .fromCatFilters()
	links_sql = NewTopLinks().fromCatFilters([]string{"test"}).fromNeuteredCatFilters(test_neutered_cats)
	rows, err = links_sql.ValidateAndExecuteRows()
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no rows")
	}

	links = []model.Link{}
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

	for _, l := range links {
		for lc := range strings.SplitSeq(l.Cats, " ") {
			for _, nc := range test_neutered_cats {
				if nc == lc {
					t.Fatalf("link %s has global cat %s", l.URL, nc)
				}
			}
		}
	}

}

func TestTopLinksWhereGlobalSummaryContains(t *testing.T) {
	// case-insensitive
	links_sql := NewTopLinks().whereGlobalSummaryContains("GoOgLe")
	rows, err := links_sql.ValidateAndExecuteRows()
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
		fromCatFilters([]string{"test"}).
		whereGlobalSummaryContains("GoOgLe").
		whereURLContains("www").
		whereURLLacks("something").
		asSignedInUser(TEST_USER_ID).
		includeNSFW().
		sortBy("newest").
		page(1).
		duringPeriod("all")
	rows, err = links_sql.ValidateAndExecuteRows()
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

func TestTopLinksWhereURLContains(t *testing.T) {
	links_sql := NewTopLinks().whereURLContains("GoOgLe")
	rows, err := links_sql.ValidateAndExecuteRows()
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
		fromCatFilters([]string{"flowers"}).
		whereURLContains("google").
		asSignedInUser(TEST_USER_ID).
		sortBy("newest")
	rows, err = links_sql.ValidateAndExecuteRows()
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

func TestTopLinksWhereURLLacks(t *testing.T) {
	// case-insensitive
	links_sql := NewTopLinks().whereURLLacks("gOoGlE")
	rows, err := links_sql.ValidateAndExecuteRows()
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
		fromCatFilters([]string{"umvc3"}).
		whereURLLacks("gOOgle").
		asSignedInUser(TEST_USER_ID).
		sortBy("newest")
	rows, err = links_sql.ValidateAndExecuteRows()
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

func TestTopLinksDuringPeriod(t *testing.T) {
	var test_periods = []struct {
		Period model.Period
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
		links_sql := NewTopLinks().duringPeriod(tp.Period)
		if tp.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tp.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		if !tp.Valid {
			continue
		}

		rows, err := links_sql.ValidateAndExecuteRows()
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		rows.Close()

		// With cats
		links_sql = links_sql.fromCatFilters([]string{"umvc3"})
		if tp.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tp.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		rows, err = links_sql.ValidateAndExecuteRows()
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		rows.Close()
	}
}

func TestTopLinksSortBy(t *testing.T) {
	var test_sorts = []struct {
		Sort  model.SortBy
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

		if !ts.Valid {
			continue
		}

		rows, err := links_sql.ValidateAndExecuteRows()
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
		case model.SortByTimesStarred:
			var last_star_count int64 = 999 // arbitrary high number
			for _, link := range links {
				if link.TimesStarred > last_star_count {
					t.Fatalf("link like count %d above previous min %d", link.TimesStarred, last_star_count)
				} else if link.TimesStarred < last_star_count {
					last_star_count = link.TimesStarred
				}
			}
		case model.SortByAverageStars:
			var last_avg_stars float32 = 999
			for _, link := range links {
				if link.AvgStars > last_avg_stars {
					t.Fatalf("link avg stars %f above previous min %f", link.AvgStars, last_avg_stars)
				} else if link.AvgStars < last_avg_stars {
					last_avg_stars = link.AvgStars
				}
			}
		case model.SortByNewest:
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
		case model.SortByOldest:
			last_date := time.Unix(0, 0) // oldest
			for _, link := range links {
				sd, err := time.Parse("2006-01-02T15:04:05Z07:00", link.SubmitDate)
				if err != nil {
					t.Fatal(err)
				}

				if sd.Before(last_date) {
					t.Fatalf("link date %s before last date %s", sd, last_date)
				} else if sd.After(last_date) {
					last_date = sd
				}
			}
		case model.SortByClicks:
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

func TestTopLinksAsSignedInUser(t *testing.T) {
	links_sql := NewTopLinks().asSignedInUser(TEST_USER_ID)
	rows, err := links_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var expected_args = []any{
		mutil.EARLIEST_STARRERS_LIMIT,
		TEST_USER_ID,
		LINKS_PAGE_LIMIT,
	}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf(
				"arg %d: got %v, want %v",
				i,
				arg,
				expected_args[i],
			)
		}
	}

	// Verify no conflict with .FromCatFilters()
	test_cats_with_spelling_variants := GetCatsOptionalPluralOrSingularForms(test_cats)
	links_sql = NewTopLinks().
		fromCatFilters(test_cats_with_spelling_variants).
		asSignedInUser(TEST_USER_ID)
	if _, err := links_sql.ValidateAndExecuteRows(); err != nil {
		t.Fatal(err)
	}

	expected_args = []any{
		mutil.EARLIEST_STARRERS_LIMIT,
		TEST_USER_ID,
		strings.Join(test_cats_with_spelling_variants, " AND "),
		LINKS_PAGE_LIMIT,
	}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf(
				"arg %d: got %v, want %v",
				i,
				arg,
				expected_args[i],
			)
		}
	}
}

func TestTopLinksIncludeNSFW(t *testing.T) {
	links_sql := NewTopLinks().includeNSFW()
	rows, err := links_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Verify no conflict with other filter methods
	links_sql = NewTopLinks().
		fromCatFilters([]string{"search", "engine", "NSFW"}).
		duringPeriod("year").
		asSignedInUser(TEST_USER_ID).
		sortBy("newest").
		page(1).
		includeNSFW()
	rows, err = links_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	id_of_test_link_having_nsfw_cats := "76"
	var l model.LinkSignedIn
	var pages int

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
			t.Fatalf(
				"got %s, want %s",
				l.ID,
				id_of_test_link_having_nsfw_cats,
			)
		}
	}

	// Verify link not present using same query without .includeNSFW()
	links_sql = NewTopLinks().
		fromCatFilters([]string{"search", "engine", "NSFW"}).
		duringPeriod("year").
		asSignedInUser(TEST_USER_ID).
		sortBy("oldest").
		page(1)
	rows, err = links_sql.ValidateAndExecuteRows()
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

func TestTopLinksPage(t *testing.T) {
	var test_cases = []struct {
		Page         uint
		WantLimitArg int
	}{
		{0, LINKS_PAGE_LIMIT},
		{1, LINKS_PAGE_LIMIT + 1},
		{2, LINKS_PAGE_LIMIT + 1},
		{3, LINKS_PAGE_LIMIT + 1},
	}

	var links_sql = NewTopLinks()

	for _, tc := range test_cases {
		links_sql = links_sql.page(tc.Page)
		if links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		}

		if tc.Page > 1 {
			limit_arg := links_sql.Args[len(links_sql.Args)-2]
			offset_arg := links_sql.Args[len(links_sql.Args)-1]

			if limit_arg != tc.WantLimitArg {
				t.Fatalf(
					"got %d, want %d with page %d",
					limit_arg,
					tc.WantLimitArg,
					tc.Page,
				)
			} else if offset_arg != (tc.Page-1)*LINKS_PAGE_LIMIT {
				t.Fatalf(
					"got %d, want %d with page %d",
					offset_arg,
					tc.WantLimitArg,
					tc.Page,
				)
			}

			continue
		}

		if links_sql.Args[len(links_sql.Args)-1] != tc.WantLimitArg {
			t.Fatalf(
				"got %d, want %d with page %d",
				links_sql.Args[len(links_sql.Args)-1],
				tc.WantLimitArg,
				tc.Page,
			)
		}
	}

	// Verify no conflict with other methods
	test_cats_with_spelling_variants := GetCatsOptionalPluralOrSingularForms(test_cats)
	links_sql = NewTopLinks().
		fromCatFilters(test_cats_with_spelling_variants).
		duringPeriod("year").
		sortBy("newest").
		asSignedInUser(TEST_USER_ID).
		includeNSFW().
		page(2)
	if _, err := links_sql.ValidateAndExecuteRows(); err != nil {
		t.Fatal(err)
	}
}

func TestTopLinksCountNSFWLinks(t *testing.T) {
	links_sql := NewTopLinks().CountNSFWLinks()
	row, err := links_sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var nsfw_links int
	if err := row.Scan(&nsfw_links); err != nil {
		t.Fatal(err)
	}

	// with NSFW params
	links_sql = NewTopLinks().
		includeNSFW().
		CountNSFWLinks()
	row, err = links_sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	if err := row.Scan(&nsfw_links); err != nil {
		t.Fatalf(
			"err: %v, sql text was %s, args were %v",
			err,
			links_sql.Text,
			links_sql.Args,
		)
	}

	// combined with other methods
	links_sql = NewTopLinks().
		fromCatFilters(test_cats).
		duringPeriod("year").
		sortBy("newest").
		asSignedInUser(TEST_USER_ID).
		CountNSFWLinks()
	if _, err := links_sql.ValidateAndExecuteRows(); err != nil {
		t.Fatalf(
			"err: %v, sql text was %s, args were %v",
			err,
			links_sql.Text,
			links_sql.Args,
		)
	}

	// MORE COMBINATIONSSSSS
	links_sql = NewTopLinks().
		sortBy("avg_stars").
		fromCatFilters(test_cats).
		whereGlobalSummaryContains("test").
		whereURLContains("www").
		whereURLLacks(".com").
		duringPeriod("all").
		asSignedInUser(TEST_USER_ID).
		includeNSFW().
		CountNSFWLinks()
	if _, err := links_sql.ValidateAndExecuteRows(); err != nil {
		t.Fatalf(
			"got error %s, sql text was %s, args were %v",
			err,
			links_sql.Text,
			links_sql.Args,
		)
	}
}
