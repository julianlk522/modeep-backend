package query

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/julianlk522/fitm/model"
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
	} else if len(cols) != 10 {
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
		{"tag_count"},
		{"like_count"},
		{"img_url"},
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

		// cats only
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

		// with period
		links_sql = links_sql.DuringPeriod("month")
		if tc.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tc.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for cats %s", tc.Cats)
		}

		// if any cats provided, args should be cat_match and limit
		// in that order
		if len(tc.Cats) == 0 || len(tc.Cats) == 1 && tc.Cats[0] == "" {
			continue
		}
		
		if len(links_sql.Args) != 2 && links_sql.Args[0] != strings.Join(tc.Cats, " ") && links_sql.Args[1] != LINKS_PAGE_LIMIT {
			t.Fatalf("got %v, want %v (should be cat_match and limit in that order)", links_sql.Args, tc.Cats)
		}

		rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		defer rows.Close()
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
		{"all", false},
		{"gobblety gook", false},
	}

	for _, tp := range test_periods {

		// period only
		links_sql := NewTopLinks().DuringPeriod(tp.Period)
		if tp.Valid && links_sql.Error != nil {
			t.Fatal(links_sql.Error)
		} else if !tp.Valid && links_sql.Error == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		defer rows.Close()

		// with cats
		// NOT a repeat of TestFromCats; testing order of method calls
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
		defer rows.Close()
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

		// scan links
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
				&link.TagCount,
				&link.LikeCount,
				&link.ImgURL,
			)
			if err != nil {
				t.Fatal(err)
			}
			links = append(links, link)
		}

		if !ts.Valid {
			continue
		}

		// verify results correctly sorted
		if ts.Sort == "rating" {
			var last_like_count int64 = 999 // arbitrary high number
			for _, link := range links {
				if link.LikeCount > last_like_count {
					t.Fatalf("link like count %d above previous min %d", link.LikeCount, last_like_count)
				} else if link.LikeCount < last_like_count {
					last_like_count = link.LikeCount
				}
			}
		} else if ts.Sort == "newest" {
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
	links_sql := NewTopLinks().AsSignedInUser(test_user_id)
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
	} else if len(cols) != 12 {
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
		{"tag_count"},
		{"like_count"},
		{"img_url"},
		{"is_liked"},
		{"is_copied"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}

	// args should be test_user_id * 2, limit
	var expected_args = []interface{}{test_user_id, test_user_id, LINKS_PAGE_LIMIT}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("arg %d: got %v, want %v", i, arg, expected_args[i])
		}
	}

	// test does not conflict with .FromCats
	links_sql = NewTopLinks().FromCats(test_cats).AsSignedInUser(test_user_id)
	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}

	// args should be test_user_id * 2, "go AND coding", limit
	expected_args = []interface{}{test_user_id, test_user_id, "go AND coding", LINKS_PAGE_LIMIT}
	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("arg %d: got %v, want %v", i, arg, expected_args[i])
		}
	}
}

func TestNSFW(t *testing.T) {
	links_sql := NewTopLinks().NSFW()
	// no opportunity for links_sql.Error to have been set

	rows, err := TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// verify does not conflict with other filter methods
	links_sql = NewTopLinks().
		FromCats([]string{"search", "engine", "NSFW"}).
		DuringPeriod("year").
		AsSignedInUser(test_user_id).
		SortBy("newest").
		Page(1).
		NSFW()

	rows, err = TestClient.Query(links_sql.Text, links_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	// verify link with ID 76 is present in results
	// (link with ID 76 is only link in test data with 'NSFW' in cats)
	var l model.LinkSignedIn
	for rows.Next() {
		if err := rows.Scan(
			&l.ID,
			&l.URL,
			&l.SubmittedBy,
			&l.SubmitDate,
			&l.Cats,
			&l.Summary,
			&l.SummaryCount,
			&l.TagCount,
			&l.LikeCount,
			&l.ImgURL,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		} else if l.ID != "76" {
			t.Fatalf("got %s, want 76", l.ID)
		}
	}

	// attempt same query without .NSFW() and verify link NOT present
	links_sql = NewTopLinks().
		FromCats([]string{"search", "engine", "NSFW"}).
		DuringPeriod("year").AsSignedInUser(test_user_id).
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
			&l.TagCount,
			&l.LikeCount,
			&l.ImgURL,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		} else if l.ID == "76" {
			t.Fatalf("got %s, want nil", l.ID)
		}
	}
}

func TestPage(t *testing.T) {
	var links_sql = NewTopLinks()

	var test_cases = []struct {
		Page int
		WantLimitArg int
	}{
		{0, LINKS_PAGE_LIMIT},
		{1, LINKS_PAGE_LIMIT + 1},
		{2, LINKS_PAGE_LIMIT + 1},
		{3, LINKS_PAGE_LIMIT + 1},
	}

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

	// ensure does not conflict with other methods
	links_sql = NewTopLinks().FromCats(test_cats).DuringPeriod("year").SortBy("newest").AsSignedInUser(test_user_id).NSFW().Page(2)

	if _, err := TestClient.Query(links_sql.Text, links_sql.Args...); err != nil {
		t.Fatal(err)
	}

	// args should be test_user_id * 2, "go AND coding", limit, offset
	// in that order
	var expected_args = []interface{}{test_user_id, test_user_id, "go AND coding", LINKS_PAGE_LIMIT + 1, LINKS_PAGE_LIMIT}

	for i, arg := range links_sql.Args {
		if arg != expected_args[i] {
			t.Fatalf("got %v, want %v", arg, expected_args[i])
		}
	}
}