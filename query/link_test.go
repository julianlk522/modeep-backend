package query

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/julianlk522/fitm/model"
	mutil "github.com/julianlk522/fitm/model/util"
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
	} else if len(cols) < 15 {
		t.Fatal("too few columns")
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
		{"like_count"},
		{"earliest_likers"},
		{"copy_count"},
		{"earliest_copiers"},
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
		links_sql := NewTopLinks().FromCats(tc.Cats)
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
		links_sql = links_sql.DuringPeriod("month", "rating")
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

func TestLinksWithURLContaining(t *testing.T) {
	links_sql := NewTopLinks().WithURLContaining("google", "newest")

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.Link
	var pages int

	for rows.Next() {
		link := model.Link{}
		err := rows.Scan(
			&link.ID,
			&link.URL,
			&link.SubmittedBy,
			&link.SubmitDate,
			&link.Cats,
			&link.Summary,
			&link.SummaryCount,
			&link.LikeCount,
			&link.EarliestLikers,
			&link.CopyCount,
			&link.EarliestCopiers,
			&link.ClickCount,
			&link.TagCount,
			&link.PreviewImgFilename,
			&pages,
		)
		if err != nil {
			t.Fatal(err)
		}
		links = append(links, link)
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
		FromCats([]string{"umvc3"}).
		WithURLContaining("google", "rating").
		AsSignedInUser(TEST_USER_ID).
		SortBy("newest")
	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		link := model.Link{}
		err := rows.Scan(
			&link.ID,
			&link.URL,
			&link.SubmittedBy,
			&link.SubmitDate,
			&link.Cats,
			&link.Summary,
			&link.SummaryCount,
			&link.LikeCount,
			&link.EarliestLikers,
			&link.CopyCount,
			&link.EarliestCopiers,
			&link.ClickCount,
			&link.TagCount,
			&link.PreviewImgFilename,
			&pages,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(links) == 0 {
		t.Fatal("no links")
	}

	for _, l := range links {
		if !strings.Contains(l.URL, "google") {
			t.Fatalf("got %s, want containing google", l.URL)
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
		links_sql := NewTopLinks().DuringPeriod(tp.Period, "newest")
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
		links_sql = links_sql.FromCats([]string{"umvc3"})
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
		{"rating", true},
		{"invalid", false},
	}

	var pages int

	for _, ts := range test_sorts {
		links_sql := NewTopLinks().SortBy(ts.Sort)
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
			link := model.Link{}
			err := rows.Scan(
				&link.ID,
				&link.URL,
				&link.SubmittedBy,
				&link.SubmitDate,
				&link.Cats,
				&link.Summary,
				&link.SummaryCount,
				&link.LikeCount,
				&link.EarliestLikers,
				&link.CopyCount,
				&link.EarliestCopiers,
				&link.ClickCount,
				&link.TagCount,
				&link.PreviewImgFilename,
				&pages,
			)
			if err != nil {
				t.Fatal(err)
			}
			links = append(links, link)
		}

		if !ts.Valid {
			continue
		}

		// Verify results correctly sorted
		switch ts.Sort {
			case "rating":
				var last_like_count int64 = 999 // arbitrary high number
				for _, link := range links {
					if link.LikeCount > last_like_count {
						t.Fatalf("link like count %d above previous min %d", link.LikeCount, last_like_count)
					} else if link.LikeCount < last_like_count {
						last_like_count = link.LikeCount
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
	} else if len(cols) != 17 {
		t.Fatal("incorrect col count")
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
		{"like_count"},
		{"earliest_likers"},
		{"copy_count"},
		{"earliest_copiers"},
		{"click_count"},
		{"tag_count"},
		{"img_file"},
		{"pages"},
		{"is_liked"},
		{"is_copied"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}

	var expected_args = []any{
		mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
		mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
		TEST_USER_ID, 
		TEST_USER_ID, 
		LINKS_PAGE_LIMIT,
	}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("arg %d: got %v, want %v", i, arg, expected_args[i])
		}
	}

	// Verify no conflict with .FromCats()
	links_sql = NewTopLinks().FromCats(test_cats).AsSignedInUser(TEST_USER_ID)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}

	// "go AND coding" modified to include plural/singular variations
	expected_args = []any{
		mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
		mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
		TEST_USER_ID, 
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
	links_sql := NewTopLinks().NSFW()
	// No opportunity for links_sql.Error to have been set

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Verify no conflict with other filter methods
	links_sql = NewTopLinks().
		FromCats([]string{"search", "engine", "NSFW"}).
		DuringPeriod("year", "rating").
		AsSignedInUser(TEST_USER_ID).
		SortBy("newest").
		Page(1).
		NSFW()

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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		} else if l.ID != id_of_test_link_having_nsfw_cats {
			t.Fatalf("got %s, want %s", l.ID, id_of_test_link_having_nsfw_cats)
		}
	}

	// Verify link not present using same query without .NSFW()
	links_sql = NewTopLinks().
		FromCats([]string{"search", "engine", "NSFW"}).
		DuringPeriod("year", "newest").
		AsSignedInUser(TEST_USER_ID).
		SortBy("newest").
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&pages,
			&l.IsLiked,
			&l.IsCopied,
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
		FromCats(test_cats).
		DuringPeriod("year", "rating").
		SortBy("newest").
		AsSignedInUser(TEST_USER_ID).
		NSFW().
		Page(2)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}

	// "go AND coding" modified to include plural/singular variations
	var expected_args = []any{
		mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT, 
		mutil.EARLIEST_LIKERS_AND_COPIERS_LIMIT,
		TEST_USER_ID, 
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
