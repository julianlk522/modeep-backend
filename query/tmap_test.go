package query

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"slices"

	"github.com/julianlk522/modeep/model"
)

const (
	TEST_LOGIN_NAME     = "jlk"
	TEST_USER_ID        = "3"
	TEST_REQ_LOGIN_NAME = "bradley"
	TEST_REQ_USER_ID    = "13"
)

var test_cats = []string{"go", "coding"}

// LINKS
func TestNewTmapSubmitted(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_REQ_LOGIN_NAME)
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var submitted_links []model.TmapLink
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
		} else if l.SubmittedBy != TEST_REQ_LOGIN_NAME {
			t.Fatalf("SubmittedBy != test login_name (%s)", TEST_REQ_LOGIN_NAME)
		} else if strings.Contains(l.Cats, "NSFW") {
			t.Fatal("should not contain NSFW in base query")
		} else if l.TagCount == 0 {
			t.Fatalf("TagCount == 0: %+v", l)
		}

		submitted_links = append(submitted_links, l)
	}	

	var submitted_ids []string
	rows, err = TestClient.Query(`SELECT id 
		FROM Links 
		WHERE submitted_by = ?
		AND global_cats NOT LIKE '%' || 'NSFW' || '%';`,
		TEST_REQ_LOGIN_NAME)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		submitted_ids = append(submitted_ids, id)
	}

	if len(submitted_ids) != len(submitted_links) {
		t.Fatalf(
			"len(submitted_ids) != len(submitted_links) (%d != %d)",
			len(submitted_ids),
			len(submitted_links),
		)
	}
}

func TestTmapSubmittedFromCatFilters(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).fromCatFilters(test_cats).Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}
	}
}

func TestTmapSubmittedFromNeuteredCatFilters(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).fromNeuteredCatFilters(test_cats).Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no results")
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

		for _, tc := range test_cats {
			for cat := range strings.SplitSeq(l.Cats, ",") {
				if cat == tc {
					t.Fatalf("got %s, should not contain %s", l.Cats, tc)
				}
			}
		}
	}

	// with .fromCatFilters()
	var neutered_cat_filters = []string{"test", "best"}
	submitted_sql = NewTmapSubmitted(TEST_LOGIN_NAME).
		fromCatFilters(test_cats).
		fromNeuteredCatFilters(neutered_cat_filters).
		Build()
	rows, err = submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no results")
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

		for _, nc := range neutered_cat_filters {
			for cat := range strings.SplitSeq(l.Cats, ",") {
				if cat == nc {
					t.Fatalf("got %s, should not contain %s", l.Cats, nc)
				}
			}
		}
	}
}

func TestTmapSubmittedAsSignedInUser(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).asSignedInUser(TEST_REQ_USER_ID).Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Verify columns
	// (should now have the StarsAssigned field)
	if rows.Next() {
		var l model.TmapLinkSignedIn
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
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapSubmittedIncludeNSFW(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_REQ_LOGIN_NAME).includeNSFW().Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}

	var found_NSFW_link bool
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
		} else if strings.Contains(l.Cats, "NSFW") {
			found_NSFW_link = true
		}
	}

	if !found_NSFW_link {
		t.Fatalf("%s's tmap does not but should contain link with NSFW tag", TEST_REQ_LOGIN_NAME)
	}
}

func TestTmapSubmittedSortBy(t *testing.T) {
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

	for _, ts := range test_sorts {
		submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).sortBy(ts.Sort).Build()
		if !ts.Valid {
			if submitted_sql.Error == nil {
				t.Fatalf("expected error for sort %s", ts.Sort)
			}
		} else {
			rows, err := submitted_sql.ValidateAndExecuteRows()
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var links []model.TmapLink
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
				
				links = append(links, l)
			}

			// Verify sorting
			switch ts.Sort {
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
				case "oldest":
					first_date := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
					for _, link := range links {
						sd, err := time.Parse("2006-01-02T15:04:05Z07:00", link.SubmitDate)
						if err != nil {
							t.Fatal(err)
						}

						if sd.Before(first_date) {
							t.Fatalf("link date %s before last date %s", sd, first_date)
						} else if sd.Before(first_date) {
							first_date = sd
						}
					}
				case "clicks":
					var highest_click_count int64 = 999
					for _, link := range links {
						if link.ClickCount > highest_click_count {
							t.Fatalf("link click count %d above previous min %d", link.ClickCount, highest_click_count)
						} else if link.ClickCount < highest_click_count {
							highest_click_count = link.ClickCount
						}
					}
			}
		}
	}
}

func TestTmapSubmittedDuringPeriod(t *testing.T) {
	var 
		submitted_links_no_period, 
		submitted_links_with_all_period, 
		submittted_links_with_week_period []model.TmapLink

	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).sortBy("newest").Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
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
		} else {
			submitted_links_no_period = append(submitted_links_no_period, l)
		}
	}

	all_period_sql := NewTmapSubmitted(TEST_LOGIN_NAME).duringPeriod("all").Build()
	rows, err = all_period_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else {
			submitted_links_with_all_period = append(submitted_links_with_all_period, l)
		}
	}

	if len(submitted_links_no_period) != len(submitted_links_with_all_period) {
		t.Fatal("submitted_links_no_period != submitted_links_with_all_period")
	}

	// verify has all the same link IDs
	var no_period_links_ids, all_period_links_ids []string
	for _, l := range submitted_links_no_period {
		no_period_links_ids = append(no_period_links_ids, l.ID)
	}

	for _, l := range submitted_links_with_all_period {
		all_period_links_ids = append(all_period_links_ids, l.ID)
	}

	for _, id := range no_period_links_ids {
		if !slices.Contains(all_period_links_ids, id) {
			t.Fatal("submitted_links_no_period != submitted_links_with_all_period")
		}
	}

	for _, id := range all_period_links_ids {
		if !slices.Contains(no_period_links_ids, id) {
			t.Fatal("submitted_links_no_period != submitted_links_with_all_period")
		}
	}

	week_period_sql := NewTmapSubmitted(TEST_LOGIN_NAME).duringPeriod("week").Build()
	rows, err = week_period_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else {
			submittted_links_with_week_period = append(submittted_links_with_week_period, l)
		}
	}

	// there should be no links submitted during last week
	if len(submittted_links_with_week_period) != 0 {
		t.Fatal("submittted_links_with_week_period != 0")
	}
}

func TestTmapSubmittedWithSummaryContaining(t *testing.T) {
	summary_snippet := "you" 
	var expected_count int
	expected_count_sql := `WITH PossibleUserSummary AS (
    SELECT
        link_id,
        text as user_summary
    FROM Summaries
    INNER JOIN Users u ON u.id = submitted_by
    WHERE u.login_name = ?
)
SELECT count(*) as link_count
FROM Links l
LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
WHERE COALESCE(pus.user_summary, l.global_summary) LIKE '%' || ? || '%'
AND l.submitted_by = ?;`
	err := TestClient.QueryRow(
		expected_count_sql, 
		TEST_LOGIN_NAME,
		summary_snippet, 
		TEST_LOGIN_NAME,
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).withSummaryContaining(summary_snippet).Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		l := model.TmapLink{}
		err := rows.Scan(
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
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, l)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

func TestTmapSubmittedWithURLContaining(t *testing.T) {
	url_snippet := "red" 
	var expected_count int
	expected_count_sql := `SELECT count(*) as link_count
		FROM Links l
		WHERE l.url LIKE '%' || ? || '%'
		AND l.submitted_by = ?;`
	err := TestClient.QueryRow(
		expected_count_sql, 
		url_snippet, 
		TEST_LOGIN_NAME,
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).withURLContaining(url_snippet).Build()
	rows, err := submitted_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		l := model.TmapLink{}
		err := rows.Scan(
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
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, l)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

// Starred
func TestNewTmapStarred(t *testing.T) {
	starred_sql := NewTmapStarred(TEST_LOGIN_NAME)
	rows, err := starred_sql.Build().ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		} else if strings.Contains(l.Cats, "NSFW") {
			t.Fatal("should not contain NSFW in base query")
		}

		// Verify tmap owner has starred
		var link_id string
		err := TestClient.QueryRow(`SELECT id
				FROM Stars
				WHERE link_id = ?
				AND user_id = ?`,
			l.ID,
			TEST_USER_ID).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapStarredFromCatFilters(t *testing.T) {
	starred_sql := NewTmapStarred(TEST_LOGIN_NAME).fromCatFilters(test_cats).Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}

		// Verify tmap owner has starred
		var link_id string
		err := TestClient.QueryRow(`SELECT id
				FROM Stars
				WHERE link_id = ?
				AND user_id = ?`,
			l.ID,
			TEST_USER_ID).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapStarredFromNeuteredCatFilters(t *testing.T) {
	starred_sql := NewTmapStarred(TEST_LOGIN_NAME).fromNeuteredCatFilters(test_cats).Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no results")
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

		for _, tc := range test_cats {
			for cat := range strings.SplitSeq(l.Cats, ",") {
				if cat == tc {
					t.Fatalf("got %s, should not contain %s", l.Cats, tc)
				}
			}
		}
	}

	// with .fromCatFilters()
	starred_sql = NewTmapStarred(TEST_LOGIN_NAME).
		fromCatFilters([]string{"test"}).
		fromNeuteredCatFilters(test_cats).
		Build()
	rows, err = starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// should also check here if there are no rows.. but need more test data
	// to provide the rows

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

		for cat := range strings.SplitSeq(l.Cats, ",") {
				for _, nc := range test_cats {
					if cat == nc {
						t.Fatalf("got %s, should not contain %s", l.Cats, nc)
					}
				}
			}
	}
}

func TestTmapStarredAsSignedInUser(t *testing.T) {
	starred_sql := NewTmapStarred(TEST_LOGIN_NAME).asSignedInUser(TEST_REQ_USER_ID).Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	links := []model.TmapLinkSignedIn{}
	for rows.Next() {
		var l model.TmapLinkSignedIn
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
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	// Verify that all starred links except those with NSFW cats are returned
	var all_starred_link_ids []string
	rows, err = TestClient.Query(`SELECT link_id
		FROM Stars
		WHERE user_id = ?`, TEST_USER_ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var link_id string
		if err := rows.Scan(&link_id); err != nil {
			t.Fatal(err)
		}
		all_starred_link_ids = append(all_starred_link_ids, link_id)
	}

	for _, lid := range all_starred_link_ids {
		var found_starred_link_in_returned_links bool
		for _, l := range links {
			if l.ID == lid {
				found_starred_link_in_returned_links = true
			}
		}

		// Verify that all non-returned starred links have NSFW cats
		if !found_starred_link_in_returned_links {
			var cats string
			if err := TestClient.QueryRow(`SELECT cats
				FROM user_cats_fts
				WHERE link_id = ?`, lid).Scan(&cats); err != nil {

				if err == sql.ErrNoRows {
					// Try global cats if no user cats found
					if err := TestClient.QueryRow(`SELECT global_cats
						FROM Links
						WHERE id = ?`, lid).Scan(&cats); err != nil {
						t.Fatal(err)
					}
				}
				t.Fatal(err)
			}

			if !slices.Contains(strings.Split(cats, ","), "NSFW") {
				t.Fatalf("non-returned link should have NSFW cats, got %s", cats)
			}
		}
	}

	// Retry with .includeNSFW() and verify that all links from
	// all_starred_link_ids are returned
	starred_sql = NewTmapStarred(TEST_LOGIN_NAME).asSignedInUser(TEST_REQ_USER_ID).includeNSFW().Build()
	rows, err = starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	links = []model.TmapLinkSignedIn{}
	for rows.Next() {
		var l model.TmapLinkSignedIn
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
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	for _, lid := range all_starred_link_ids {
		var found_starred_link_in_returned_links bool
		for _, l := range links {
			if l.ID == lid {
				found_starred_link_in_returned_links = true
			}
		}
		if !found_starred_link_in_returned_links {
			t.Fatalf("non-returned link found despite enabled NSFW flag: %s", lid)
		}
	}
}

func TestTmapStarredIncludeNSFW(t *testing.T) {
	starred_sql := NewTmapStarred(TEST_LOGIN_NAME).includeNSFW().Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var found_NSFW_link bool
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
		} else if strings.Contains(l.Cats, "NSFW") {
			found_NSFW_link = true
		}
	}

	if !found_NSFW_link {
		t.Fatalf("%s's tmap does not but should contain 1 starred link with NSFW tag", TEST_LOGIN_NAME)
	}
}

func TestTmapStarredDuringPeriod(t *testing.T) {
	var starred_no_period, starred_period_all, starred_period_week []model.TmapLink
	starred_sql := NewTmapStarred(TEST_LOGIN_NAME).Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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

		starred_no_period = append(starred_no_period, l)
	}

	starred_sql = NewTmapStarred(TEST_LOGIN_NAME).duringPeriod("all").Build()
	rows, err = starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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

		starred_period_all = append(starred_period_all, l)
	}

	if len(starred_no_period) != len(starred_period_all) {
		t.Fatal("starred_no_period != starred_period_all")
	}

	starred_sql = NewTmapStarred(TEST_LOGIN_NAME).duringPeriod("week").Build()
	rows, err = starred_sql.ValidateAndExecuteRows()
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

		starred_period_week = append(starred_period_week, l)
	}

	if len(starred_period_week) != 0 {
		t.Fatal("should be no links starred within last week")
	}
}

func TestTmapStarredWithSummaryContaining(t *testing.T) {
	summary_snippet := "you" 
	var expected_count int
	expected_count_sql := `WITH PossibleUserSummary AS (
		SELECT
			link_id,
			text as user_summary
		FROM Summaries
		INNER JOIN Users u ON u.id = submitted_by
		WHERE u.login_name = ?
	)
	SELECT count(*) as link_count
	FROM Links l
	LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
	INNER JOIN Users u ON u.login_name = ?
	INNER JOIN Stars s ON s.link_id = l.id AND s.user_id = u.id
	WHERE COALESCE(pus.user_summary, l.global_summary) LIKE '%' || ? || '%';`
	err := TestClient.QueryRow(
			expected_count_sql,
			TEST_LOGIN_NAME, 
			TEST_LOGIN_NAME,
			summary_snippet, 
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	starred_sql := NewTmapSubmitted(TEST_LOGIN_NAME).withSummaryContaining(summary_snippet).Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		l := model.TmapLink{}
		err := rows.Scan(
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
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, l)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

func TestTmapStarredWithURLContaining(t *testing.T) {
	url_snippet := "coding" 
	var expected_count int
	expected_count_sql := `SELECT count(*) as times_starred
		FROM Stars s
		LEFT JOIN Users u ON u.id = s.user_id
		LEFT JOIN Links l ON l.id = s.link_id
		WHERE l.url LIKE '%' || ? || '%'
		AND s.user_id = ?;`
	err := TestClient.QueryRow(
			expected_count_sql, 
			url_snippet, 
			TEST_LOGIN_NAME,
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	starred_sql := NewTmapSubmitted(TEST_LOGIN_NAME).withURLContaining(url_snippet).Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		l := model.TmapLink{}
		err := rows.Scan(
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
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, l)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

func TestNewTmapTagged(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME)
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}

		// Verify tmap owner has tagged
		var link_id string
		err := TestClient.QueryRow(`SELECT id
				FROM Tags
				WHERE link_id = ?
				AND submitted_by = ?;`,
			l.ID,
			TEST_LOGIN_NAME).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapTaggedFromCatFilters(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).fromCatFilters(test_cats).Build()
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}

		// Verify tmap owner has tagged
		var link_id string
		err := TestClient.QueryRow(`SELECT id
			FROM Tags
			WHERE link_id = ?
			AND submitted_by = ?`,
			l.ID,
			TEST_LOGIN_NAME,
		).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapTaggedFromNeuteredCatFilters(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).fromNeuteredCatFilters(test_cats).Build()
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no results")
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

		for _, tc := range test_cats {
			for cat := range strings.SplitSeq(l.Cats, ",") {
				if cat == tc {
					t.Fatalf("got %s, should not contain %s", l.Cats, tc)
				}
			}
		}
	}

	// with .fromCatFilters()
	tagged_sql = NewTmapTagged(TEST_LOGIN_NAME).
		fromCatFilters([]string{"test"}).
		fromNeuteredCatFilters(test_cats).
		Build()
	rows, err = tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no results")
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

		for cat := range strings.SplitSeq(l.Cats, ",") {
				for _, tc := range test_cats {
					if cat == tc {
						t.Fatalf("got %s, should not contain %s", l.Cats, tc)
					}
				}
			}
	}
}

func TestTmapTaggedAsSignedInUser(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).asSignedInUser(TEST_REQ_USER_ID).Build()
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Verify columns
	if rows.Next() {
		var l model.TmapLinkSignedIn
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
			&l.StarsAssigned,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapTaggedIncludeNSFW(t *testing.T) {
	starred_sql := NewTmapTagged(TEST_LOGIN_NAME).includeNSFW().Build()
	rows, err := starred_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var found_NSFW_link bool
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
		} else if strings.Contains(l.Cats, "NSFW") {
			found_NSFW_link = true
		}
	}

	if !found_NSFW_link {
		t.Fatalf("%s's tmap does not but should contain 1 tagged link with NSFW tag", TEST_LOGIN_NAME)
	}
}

func TestTmapTaggedDuringPeriod(t *testing.T) {
	var tagged_no_period, tagged_period_all, tagged_period_week []model.TmapLink
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).Build()
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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

		tagged_no_period = append(tagged_no_period, l)
	}

	tagged_sql = NewTmapTagged(TEST_LOGIN_NAME).duringPeriod("all").Build()
	rows, err = tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

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

		tagged_period_all = append(tagged_period_all, l)
	}

	if len(tagged_no_period) != len(tagged_period_all) {
		t.Fatal("tagged_no_period != tagged_period_all")
	}

	tagged_sql = NewTmapTagged(TEST_LOGIN_NAME).duringPeriod("week").Build()
	rows, err = tagged_sql.ValidateAndExecuteRows()
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

		tagged_period_week = append(tagged_period_week, l)
	}

	if len(tagged_period_week) != 0 {
		t.Fatal("should be no links tagged within last week")
	}
}

func TestTmapTaggedWithSummaryContaining(t *testing.T) {
	summary_snippet := "test"
	var expected_count int
	expected_count_sql := `WITH PossibleUserSummary AS (
		SELECT
			link_id,
			text as user_summary
		FROM Summaries
		INNER JOIN Users u ON u.id = submitted_by
		WHERE u.login_name = ?
	),
	UserStars AS (
		SELECT s.link_id
		FROM Stars s
		INNER JOIN Users u ON u.id = s.user_id
		WHERE u.login_name = ?
	)
	SELECT count(*) as link_count
	FROM Links l
	LEFT JOIN PossibleUserSummary pus ON l.id = pus.link_id
	INNER JOIN Tags t ON t.link_id = l.id AND t.submitted_by = ?
	WHERE COALESCE(pus.user_summary, l.global_summary) LIKE '%' || ? || '%'
	AND l.submitted_by != ?
	AND l.id NOT IN (SELECT link_id FROM UserStars);`
	err := TestClient.QueryRow(
			expected_count_sql, 
			TEST_LOGIN_NAME,
			TEST_LOGIN_NAME,
			TEST_LOGIN_NAME,
			summary_snippet,
			TEST_LOGIN_NAME,
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).withSummaryContaining(summary_snippet).Build()
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		l := model.TmapLink{}
		err := rows.Scan(
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
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, l)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

func TestTmapTaggedWithURLContaining(t *testing.T) {
	url_snippet := "cars"
	var expected_count int
	expected_count_sql := `WITH UserCats AS (
		SELECT link_id, cats as user_cats
		FROM user_cats_fts
		WHERE submitted_by = ?
	),
	UserStars AS (
		SELECT s.link_id
		FROM Stars s
		INNER JOIN Users u ON u.id = s.user_id
		WHERE u.login_name = ?
	)
	SELECT count(l.id) AS count
	FROM Links l
	INNER JOIN UserCats uct ON l.id = uct.link_id
	LEFT JOIN UserStars us ON l.id = us.link_id
	WHERE l.id NOT IN (
			SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
	)
	AND l.submitted_by != ?
	AND l.id NOT IN
			(SELECT link_id FROM UserStars)
	AND l.url LIKE '%' || ? || '%'
	ORDER BY l.id DESC;`
	err := TestClient.QueryRow(
			expected_count_sql, 
			TEST_LOGIN_NAME,
			TEST_LOGIN_NAME,
			TEST_LOGIN_NAME,
			url_snippet,
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).withURLContaining(url_snippet).Build()
	rows, err := tagged_sql.ValidateAndExecuteRows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		l := model.TmapLink{}
		err := rows.Scan(
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
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, l)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

// NSFW LINKS COUNT
func TestNewTmapNSFWLinksCount(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_LOGIN_NAME)
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = row.Scan(&count); err != nil {
		t.Fatal(err)
	}

	// (submitted, starred, or tagged links with global_cat "NSFW"
	// OR where user's tag contains "NSFW")
	var expected_count int
	sql_manual := `WITH PossibleUserCatsNSFW AS (
    SELECT
        link_id,
        cats AS user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
        AND cats MATCH 'NSFW'
),
GlobalNSFWCats AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH 'NSFW'
),
UserStars AS (
    SELECT s.link_id
    FROM Stars s
    INNER JOIN Users u ON u.id = s.user_id
    WHERE u.login_name = ?
)
SELECT count(*) as NSFW_link_count
FROM Links l
LEFT JOIN PossibleUserCatsNSFW pucnsfw ON l.id = pucnsfw.link_id
LEFT JOIN GlobalNSFWCats gnsfwc ON l.id = gnsfwc.link_id
WHERE
    (gnsfwc.global_cats IS NOT NULL OR pucnsfw.user_cats IS NOT NULL)
AND (
	l.submitted_by = ?
	OR l.id IN (SELECT link_id FROM UserStars)
	OR l.id IN
		(
		SELECT link_id
		FROM PossibleUserCatsNSFW
		)
	);`

	if err := TestClient.QueryRow(
		sql_manual,
		TEST_LOGIN_NAME, TEST_LOGIN_NAME, TEST_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	} else if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}

	// .FromCatFilters()
	test_cat_filters := GetCatsOptionalPluralOrSingularForms(
		[]string{"engine", "search"},
	)
	sql = sql.fromCatFilters(test_cat_filters)
	row, err = sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}

	sql_manual = `WITH PossibleUserCatsNSFW AS (
	SELECT
		link_id,
		cats AS user_cats
	FROM user_cats_fts
	WHERE submitted_by = ?
		AND cats MATCH 'NSFW'
),
GlobalNSFWCats AS (
	SELECT
			link_id,
			global_cats
	FROM global_cats_fts
	WHERE global_cats MATCH 'NSFW'
),
PossibleUserCatsMatchingRequestParams AS (
	SELECT
			link_id,
			cats AS user_cats
	FROM user_cats_fts
	WHERE submitted_by = ?
	AND cats MATCH ?
),
GlobalCatsMatchingRequestParams AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH ?
),
UserStars AS (
	SELECT s.link_id
	FROM Stars s
	INNER JOIN Users u ON u.id = s.user_id
	WHERE u.login_name = ?
)
SELECT count(*) as NSFW_link_count
FROM Links l
LEFT JOIN PossibleUserCatsNSFW pucnsfw ON l.id = pucnsfw.link_id
LEFT JOIN GlobalNSFWCats gnsfwc ON l.id = gnsfwc.link_id
LEFT JOIN PossibleUserCatsMatchingRequestParams pucmrp ON l.id = pucmrp.link_id
LEFT JOIN GlobalCatsMatchingRequestParams gcmrp ON l.id = gcmrp.link_id
WHERE
		(gnsfwc.global_cats IS NOT NULL
		OR
		pucnsfw.user_cats IS NOT NULL)
AND (gcmrp.global_cats IS NOT NULL OR pucmrp.user_cats IS NOT NULL)
AND (
	l.submitted_by = ?
	OR l.id IN (SELECT link_id FROM UserStars)
	OR l.id IN
			(
			SELECT link_id
			FROM PossibleUserCatsNSFW
			)
	);`
	if err := TestClient.QueryRow(
		sql_manual,
		TEST_LOGIN_NAME, 
		TEST_LOGIN_NAME,
		strings.Join(test_cat_filters, " AND "),
		strings.Join(test_cat_filters, " AND "),
		TEST_LOGIN_NAME,
		TEST_LOGIN_NAME,
	).Scan(&expected_count); err != nil { 
		t.Fatal(err)
	} else if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountFromOptions(t *testing.T) {
	var test_cat_filters = []string{
		"search",
		"engine",
	}
	// cat filters are always first expanded into optional singular/plural
	// forms by GetTmapOptsFromRequestParams()
	test_cat_filters = GetCatsOptionalPluralOrSingularForms(test_cat_filters)

	var test_options_and_expected_counts = []struct {
		Options model.TmapNSFWLinksCountOptions
		Valid bool
	}{
		{
			model.TmapNSFWLinksCountOptions{
				CatFiltersWithSpellingVariants: test_cat_filters,
				Period: "all",
				URLContains: "googler",
			}, 
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				CatFiltersWithSpellingVariants: test_cat_filters,
				Period: "all",
				URLContains: "not_googler",
			}, 
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				CatFiltersWithSpellingVariants: test_cat_filters,
				Period: "all",
			},
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "submitted",
				CatFiltersWithSpellingVariants: test_cat_filters,
			},
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "tagged",
				CatFiltersWithSpellingVariants: test_cat_filters, 
			},
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "starred",
				CatFiltersWithSpellingVariants: test_cat_filters,
			},
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "boop",
				CatFiltersWithSpellingVariants: test_cat_filters, 
			},
			false,
		},
	}

	for _, test := range test_options_and_expected_counts {
		sql := NewTmapNSFWLinksCount(TEST_LOGIN_NAME).FromOptions(&test.Options)
		row, err := sql.ValidateAndExecuteRow()
		if (test.Valid && err != nil) || (!test.Valid && err == nil) {
			t.Fatalf("expected %t, got %t", test.Valid, err == nil)
		}

		if test.Valid {
			var count int
			if err = row.Scan(&count); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestTmapNSFWLinksCountSubmittedOnly(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).submittedOnly() 
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = row.Scan(&count); err != nil {
		t.Fatal(err)
	}

	sql_manual := `WITH PossibleUserCatsNSFW AS (
    SELECT
        link_id,
        cats AS user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
        AND cats MATCH 'NSFW'
),
GlobalNSFWCats AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH 'NSFW'
)
SELECT count(*) as NSFW_link_count
FROM Links l
LEFT JOIN PossibleUserCatsNSFW pucnsfw ON l.id = pucnsfw.link_id
LEFT JOIN GlobalNSFWCats gnsfwc ON l.id = gnsfwc.link_id
WHERE (gnsfwc.global_cats IS NOT NULL OR pucnsfw.user_cats IS NOT NULL)
AND l.submitted_by = ?;`

	var expected_count int
	if err := TestClient.QueryRow(
		sql_manual,
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	} else if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountStarredOnly(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).starredOnly()
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}

	var expected_count int
	sql_manual := `WITH PossibleUserCatsNSFW AS (
    SELECT
        link_id,
        cats AS user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
        AND cats MATCH 'NSFW'
),
GlobalNSFWCats AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH 'NSFW'
),
UserStars AS (
    SELECT s.link_id
    FROM Stars s
    INNER JOIN Users u ON u.id = s.user_id
    WHERE u.login_name = ?
)
SELECT count(*) as NSFW_link_count
FROM Links l
LEFT JOIN PossibleUserCatsNSFW pucnsfw ON l.id = pucnsfw.link_id
LEFT JOIN GlobalNSFWCats gnsfwc ON l.id = gnsfwc.link_id
WHERE
    (gnsfwc.global_cats IS NOT NULL OR pucnsfw.user_cats IS NOT NULL)
AND l.id IN (SELECT link_id FROM UserStars);`
	if err := TestClient.QueryRow(
		sql_manual, 
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	}

	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountTaggedOnly(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).taggedOnly()
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}

	var expected_count int
	sql_manual := `WITH PossibleUserCatsNSFW AS (
    SELECT
        link_id,
        cats AS user_cats
    FROM user_cats_fts
    WHERE submitted_by = ?
        AND cats MATCH 'NSFW'
),
GlobalNSFWCats AS (
    SELECT
        link_id,
        global_cats
    FROM global_cats_fts
    WHERE global_cats MATCH 'NSFW'
),
UserStars AS (
    SELECT s.link_id
    FROM Stars s
    INNER JOIN Users u ON u.id = s.user_id
    WHERE u.login_name = ?
)
SELECT count(*) as NSFW_link_count
FROM Links l
LEFT JOIN PossibleUserCatsNSFW pucnsfw ON l.id = pucnsfw.link_id
LEFT JOIN GlobalNSFWCats gnsfwc ON l.id = gnsfwc.link_id
WHERE (gnsfwc.global_cats IS NOT NULL OR pucnsfw.user_cats IS NOT NULL)
AND l.submitted_by != ?
AND l.id NOT IN
	(SELECT link_id FROM UserStars);`

	if err := TestClient.QueryRow(
		sql_manual, 
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	} else if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountDuringPeriod(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME)
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var total_count int
	if err := row.Scan(&total_count); err != nil {
		t.Fatal(err)
	}

	// should equal "all" period
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).duringPeriod("all")
	row, err = sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count_during_all_period int
	if err := row.Scan(&count_during_all_period); err != nil {
		t.Fatal(err)
	}

	if total_count != count_during_all_period {
		t.Fatalf("expected %d, got %d", total_count, count_during_all_period)
	}

	// last week (none)
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).duringPeriod("week")
	row, err = sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count_during_last_week int
	if err := row.Scan(&count_during_last_week); err != nil {
		t.Fatal(err)
	}

	if count_during_last_week != 0 {
		t.Fatalf("expected %d, got %d", 0, count_during_last_week)
	}
}

func TestTmapNSFWLinksCountWithSummaryContaining(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).withSummaryContaining("web")
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}

	var expected_count int
	nsfw_links_sql := `SELECT COUNT(DISTINCT L.id) as nsfw_links
FROM Links L
LEFT JOIN Users U ON L.submitted_by = U.login_name
LEFT JOIN Stars S ON S.user_id = U.id
LEFT JOIN Tags T ON T.submitted_by = L.submitted_by
WHERE L.global_cats LIKE '%' || 'NSFW' || '%'
	AND L.global_summary LIKE '%' || ? || '%'
  	AND (L.submitted_by = ? OR T.submitted_by = ? OR U.login_name = ?);`
	if err := TestClient.QueryRow(
		nsfw_links_sql,
		"web",
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	} else if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountWithURLContaining(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME)
	row, err := sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var overall_count int
	if err := row.Scan(&overall_count); err != nil {
		t.Fatal(err)
	}

	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).withURLContaining("googler")
	row, err = sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count_with_url_containing int
	if err := row.Scan(&count_with_url_containing); err != nil {
		t.Fatal(err)
	}

	if overall_count != count_with_url_containing {
		t.Fatalf("expected %d, got %d", overall_count, count_with_url_containing)
	}

	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).withURLContaining("not_googler")
	row, err = sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var count_with_url_not_containing int
	if err := row.Scan(&count_with_url_not_containing); err != nil {
		t.Fatal(err)
	}

	if count_with_url_not_containing != 0 {
		t.Fatalf("expected %d, got %d", 0, count_with_url_not_containing)
	}
}

// PROFILE
func TestNewTmapProfile(t *testing.T) {
	profile_sql := NewTmapProfile(TEST_LOGIN_NAME)
	row, err := profile_sql.ValidateAndExecuteRow()
	if err != nil {
		t.Fatal(err)
	}
	var profile model.Profile
	if err := row.Scan(
		&profile.LoginName,
		&profile.PFP,
		&profile.About,
		&profile.Email,
		&profile.CreatedAt,
	); err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
}