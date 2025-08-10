package query

import (
	"database/sql"
	"strings"
	"testing"

	"slices"

	"github.com/julianlk522/fitm/model"
)

const (
	TEST_LOGIN_NAME     = "jlk"
	TEST_USER_ID        = "3"
	TEST_REQ_LOGIN_NAME = "bradley"
	TEST_REQ_USER_ID    = "13"
)

var test_cats = []string{"go", "coding"}

func TestNewTmapProfile(t *testing.T) {
	profile_sql := NewTmapProfile(TEST_LOGIN_NAME)

	var profile model.Profile
	if err := TestClient.QueryRow(profile_sql.Text, profile_sql.Args...).Scan(
		&profile.LoginName,
		&profile.PFP,
		&profile.About,
		&profile.Email,
		&profile.Created,
	); err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
}

func TestNewTmapNSFWLinksCount(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_LOGIN_NAME)
	var count int
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// Copied / Tagged
	// test user jlk copied link 76 with global tag "engine,search,NSFW",
	// test user jlk tagged link c880180f-935d-4fd1-9a82-14dca4bd18f3 with
	// cat "NSFW"
	// (count should be 2)

	expected_count := 2
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}

	// .FromCats()
	sql = sql.FromCats([]string{"engine", "search"})
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// Only test link 76 has cats "engine" and "search" in addition to "NSFW"
	// (count should be 1)
	expected_count = 1
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}

	// Submitted
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME)
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// Only link test_req_login_name (bradley) has submitted with cat "NSFW" is 76
	// (count should be 1)
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountSubmittedOnly(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).SubmittedOnly()
	var count int
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	var expected_count int
	nsfw_submitted_links_sql := `SELECT count(*) as nsfw_submitted_links 
		FROM LINKS 
		WHERE submitted_by = ? 
		AND global_cats LIKE '%' || 'NSFW' || '%';`
	if err := TestClient.QueryRow(
		nsfw_submitted_links_sql, 
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)	
	}

	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountCopiedOnly(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).CopiedOnly()
	var count int
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	var expected_count int
	nsfw_copied_links_sql := `SELECT count(*) as copy_count
		FROM "Link Copies" lc
		LEFT JOIN Users u ON u.id = lc.user_id
		LEFT JOIN Links l ON l.id = lc.link_id
		WHERE lc.user_id = ?
		AND l.global_cats LIKE '%' || 'NSFW' || '%';`
	if err := TestClient.QueryRow(
		nsfw_copied_links_sql, 
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	}

	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountTaggedOnly(t *testing.T) {
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).TaggedOnly()
	var count int
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	var expected_count int
	nsfw_tagged_links_sql := `SELECT count(*) as tag_count
		FROM Tags t
		LEFT JOIN Links l ON t.link_id = l.id
		WHERE t.submitted_by = ?
		AND l.submitted_by != ?
		AND t.cats LIKE '%' || 'NSFW' || '%';`
	if err := TestClient.QueryRow(
		nsfw_tagged_links_sql, 
		TEST_REQ_LOGIN_NAME,
		TEST_REQ_LOGIN_NAME,
	).Scan(&expected_count); err != nil {
		t.Fatal(err)
	}

	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestTmapNSFWLinksCountDuringPeriod(t *testing.T) {
	// get NSFW links count overall first:
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME)
	var total_countut int
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&total_countut); err != nil {
		t.Fatal(err)
	}

	// should equal "all" period
	var count_during_all_period int
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).DuringPeriod("all")
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count_during_all_period); err != nil {
		t.Fatal(err)
	}

	if total_countut != count_during_all_period {
		t.Fatalf("expected %d, got %d", total_countut, count_during_all_period)
	}

	// last week (none)
	var count_during_last_week int
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).DuringPeriod("week")
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count_during_last_week); err != nil {
		t.Fatal(err)
	}

	if count_during_last_week != 0 {
		t.Fatalf("expected %d, got %d", 0, count_during_last_week)
	}
}

func TestTmapNSFWLinksCountWithURLContaining(t *testing.T) {
	// user bradley has 1 NSFW tmap link: "https://www.googler.com/"
	// count should be 1 overall and 0 with URL contains: {anything not in that}

	var overall_count int
	sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME)
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&overall_count); err != nil {
		t.Fatal(err)
	}

	var count_with_url_containing int
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).WithURLContaining("googler")
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count_with_url_containing); err != nil {
		t.Fatal(err)
	}

	if overall_count != count_with_url_containing {
		t.Fatalf("expected %d, got %d", overall_count, count_with_url_containing)
	}

	var count_with_url_not_containing int
	sql = NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).WithURLContaining("not_googler")
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count_with_url_not_containing); err != nil {
		t.Fatal(err)
	}

	if count_with_url_not_containing != 0 {
		t.Fatalf("expected %d, got %d", 0, count_with_url_not_containing)
	}
}

func TestTmapNSFWLinksCountFromOptions(t *testing.T) {
	var test_options_and_expected_counts = []struct {
		Options model.TmapNSFWLinksCountOptions
		ExpectedCount int
		Valid bool
	}{
		{
			model.TmapNSFWLinksCountOptions{
				CatsFilter: []string{
					"search",
					"engine",
				},
				Period: "all",
				URLContains: "googler",
			}, 
			1,
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				CatsFilter: []string{
					"search",
					"engine",
				},
				Period: "all",
				URLContains: "not_googler",
			}, 
			0,
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				CatsFilter: []string{
					"search",
					"engine",
				},
				Period: "all",
			}, 
			1,
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "submitted",
				CatsFilter: []string{
					"search",
					"engine",
				},
			}, 
			1,
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "tagged",
				CatsFilter: []string{
					"search",
					"engine",
				},
			}, 
			0,
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "copied",
				CatsFilter: []string{
					"search",
					"engine",
				},
			}, 
			0,
			true,
		},
		{
			model.TmapNSFWLinksCountOptions{
				OnlySection: "boop",
				CatsFilter: []string{
					"search",
					"engine",
				},
			}, 
			1,
			false,
		},
	}

	for _, test := range test_options_and_expected_counts {
		sql := NewTmapNSFWLinksCount(TEST_REQ_LOGIN_NAME).FromOptions(&test.Options)
		var count int
		if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
			t.Fatal(err)
		}

		if (test.Valid && sql.Error != nil) || (!test.Valid && sql.Error == nil) {
			if sql.Error == nil {
				t.Fatalf("expected error, got nil")
			}
		}

		if count != test.ExpectedCount {
			t.Fatalf("expected %d, got %d (opts: %+v)", 
				test.ExpectedCount, 
				count,
				test.Options,
			)
		}
	}
	
}

func TestNewTmapSubmitted(t *testing.T) {
	// Retrieve all IDs of links submitted by user
	var submitted_ids []string
	rows, err := TestClient.Query(`SELECT id 
		FROM Links 
		WHERE submitted_by = ?
		AND global_cats NOT LIKE '%' || 'NSFW' || '%';`, // exclude NSFW in base query
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

	// Verify all submitted links are present after executing query
	submitted_sql := NewTmapSubmitted(TEST_REQ_LOGIN_NAME)
	if submitted_sql.Error != nil {
		t.Fatal(submitted_sql.Error)
	}

	rows, err = TestClient.Query(submitted_sql.Text, submitted_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		} else if l.SubmittedBy != TEST_REQ_LOGIN_NAME {
			t.Fatalf("SubmittedBy != test login_name (%s)", TEST_REQ_LOGIN_NAME)
		} else if l.TagCount == 0 {
			t.Fatalf("TagCount == 0: %+v", l)
		} else if strings.Contains(l.Cats, "NSFW") {
			t.Fatal("should not contain NSFW in base query")
		}

		// Remove from submitted_ids if returned by query
		for i := 0; i < len(submitted_ids); i++ {
			if l.ID == submitted_ids[i] {
				submitted_ids = append(submitted_ids[0:i], submitted_ids[i+1:]...)
				break
			}
		}
	}

	// If any IDs are left in submitted_ids then they were incorrectly
	// omitted by query
	if len(submitted_ids) > 0 {
		t.Fatalf("not all submitted links returned, see missing IDs: %+v", submitted_ids)
	}
}

func TestTmapSubmittedFromCats(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).FromCats(test_cats)
	if submitted_sql.Error != nil {
		t.Fatal(submitted_sql.Error)
	}

	rows, err := TestClient.Query(submitted_sql.Text, submitted_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

func TestTmapSubmittedAsSignedInUser(t *testing.T) {
	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).AsSignedInUser(TEST_REQ_USER_ID)
	if submitted_sql.Error != nil {
		t.Fatal(submitted_sql.Error)
	}

	rows, err := TestClient.Query(submitted_sql.Text, submitted_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapSubmittedNSFW(t *testing.T) {
	submitted_sql := NewTmapSubmitted("bradley").NSFW()
	rows, err := TestClient.Query(submitted_sql.Text, submitted_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	// Verify TEST_LOGIN_NAME's tmap contains link with NSFW tag
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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
		t.Fatal("bradley's tmap does not but should contain link with NSFW tag")
	}
}

func TestTmapSubmittedDuringPeriod(t *testing.T) {
	var 
		submitted_links_no_period, 
		submitted_links_with_all_period, 
		submittted_links_with_week_period []model.TmapLink

	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).SortBy("newest")
	rows, _ := TestClient.Query(submitted_sql.Text, submitted_sql.Args...)

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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		} else {
			submitted_links_no_period = append(submitted_links_no_period, l)
		}
	}

	all_period_sql := NewTmapSubmitted(TEST_LOGIN_NAME).DuringPeriod("all")
	rows, _ = TestClient.Query(all_period_sql.Text, all_period_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

	week_period_sql := NewTmapSubmitted(TEST_LOGIN_NAME).DuringPeriod("week")
	rows, err := TestClient.Query(week_period_sql.Text, week_period_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

	submitted_sql := NewTmapSubmitted(TEST_LOGIN_NAME).WithURLContaining(url_snippet)
	rows, err := TestClient.Query(submitted_sql.Text, submitted_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		link := model.TmapLink{}
		err := rows.Scan(
			&link.ID,
			&link.URL,
			&link.SubmittedBy,
			&link.SubmitDate,
			&link.Cats,
			&link.CatsFromUser,
			&link.Summary,
			&link.SummaryCount,
			&link.LikeCount,
			&link.EarliestLikers,
			&link.CopyCount,
			&link.EarliestCopiers,
			&link.ClickCount,
			&link.TagCount,
			&link.PreviewImgFilename,
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, link)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

// Copied
func TestNewTmapCopied(t *testing.T) {
	copied_sql := NewTmapCopied(TEST_LOGIN_NAME)
	if copied_sql.Error != nil {
		t.Fatal(copied_sql.Error)
	}

	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

		// Verify tmap owner has copied
		var link_id string
		err := TestClient.QueryRow(`SELECT id
				FROM "Link Copies"
				WHERE link_id = ?
				AND user_id = ?`,
			l.ID,
			TEST_USER_ID).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapCopiedFromCats(t *testing.T) {
	copied_sql := NewTmapCopied(TEST_LOGIN_NAME).FromCats(test_cats)
	if copied_sql.Error != nil {
		t.Fatal(copied_sql.Error)
	}

	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

		// Verify tmap owner has copied
		var link_id string
		err := TestClient.QueryRow(`SELECT id
				FROM "Link Copies"
				WHERE link_id = ?
				AND user_id = ?`,
			l.ID,
			TEST_USER_ID).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapCopiedAsSignedInUser(t *testing.T) {
	copied_sql := NewTmapCopied(TEST_LOGIN_NAME).AsSignedInUser(TEST_REQ_USER_ID)
	if copied_sql.Error != nil {
		t.Fatal(copied_sql.Error)
	}

	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Scan links, verifying columns
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	// Manually search Link Copies table to verify that all copied links,
	// EXCEPT those with NSFW cats, are returned
	var all_copied_link_ids []string
	rows, err = TestClient.Query(`SELECT link_id
		FROM "Link Copies"
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
		all_copied_link_ids = append(all_copied_link_ids, link_id)
	}

	for _, lid := range all_copied_link_ids {
		var found_copied_link_in_returned_links bool
		for _, l := range links {
			if l.ID == lid {
				found_copied_link_in_returned_links = true
			}
		}

		// Verify that all non-returned copied links have NSFW cats
		if !found_copied_link_in_returned_links {
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

	// Retry with .NSFW() and verify that _all_ links from all_copied_link_ids
	// are returned
	copied_sql = NewTmapCopied(TEST_LOGIN_NAME).AsSignedInUser(TEST_REQ_USER_ID).NSFW()
	rows, err = TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}

		links = append(links, l)
	}

	for _, lid := range all_copied_link_ids {
		var found_copied_link_in_returned_links bool
		for _, l := range links {
			if l.ID == lid {
				found_copied_link_in_returned_links = true
			}
		}
		if !found_copied_link_in_returned_links {
			t.Fatalf("non-returned link found despite enabled NSFW flag: %s", lid)
		}
	}
}

func TestTmapCopiedNSFW(t *testing.T) {
	// TEST_LOGIN_NAME (jlk) has copied 1 link with NSFW tag
	copied_sql := NewTmapCopied(TEST_LOGIN_NAME).NSFW()
	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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
		t.Fatal("jlk's tmap does not but should contain 1 copied link with NSFW tag")
	}
}

func TestTmapCopiedDuringPeriod(t *testing.T) {
	var copied_no_period, copied_period_all, copied_period_week []model.TmapLink
	copied_sql := NewTmapCopied(TEST_LOGIN_NAME)
	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		}

		copied_no_period = append(copied_no_period, l)
	}

	copied_sql = NewTmapCopied(TEST_LOGIN_NAME).DuringPeriod("all")
	rows, err = TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		}

		copied_period_all = append(copied_period_all, l)
	}

	if len(copied_no_period) != len(copied_period_all) {
		t.Fatal("copied_no_period != copied_period_all")
	}

	copied_sql = NewTmapCopied(TEST_LOGIN_NAME).DuringPeriod("week")
	rows, err = TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		}

		copied_period_week = append(copied_period_week, l)
	}

	if len(copied_period_week) != 0 {
		t.Fatal("should be no links copied within last week")
	}
}

func TestTmapCopiedWithURLContaining(t *testing.T) {
	url_snippet := "coding" 
	var expected_count int
	expected_count_sql := `SELECT count(*) as copy_count
		FROM "Link Copies" lc
		LEFT JOIN Users u ON u.id = lc.user_id
		LEFT JOIN Links l ON l.id = lc.link_id
		WHERE l.url LIKE '%' || ? || '%'
		AND lc.user_id = ?;`
	err := TestClient.QueryRow(
			expected_count_sql, 
			url_snippet, 
			TEST_LOGIN_NAME,
		).Scan(&expected_count)
	if err != nil {
		t.Fatal(err)
	}

	copied_sql := NewTmapSubmitted(TEST_LOGIN_NAME).WithURLContaining(url_snippet)
	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		link := model.TmapLink{}
		err := rows.Scan(
			&link.ID,
			&link.URL,
			&link.SubmittedBy,
			&link.SubmitDate,
			&link.Cats,
			&link.CatsFromUser,
			&link.Summary,
			&link.SummaryCount,
			&link.LikeCount,
			&link.EarliestLikers,
			&link.CopyCount,
			&link.EarliestCopiers,
			&link.ClickCount,
			&link.TagCount,
			&link.PreviewImgFilename,
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, link)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

func TestNewTmapTagged(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME)
	if tagged_sql.Error != nil {
		t.Fatal(tagged_sql.Error)
	}

	rows, err := TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

func TestTmapTaggedFromCats(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).FromCats(test_cats)
	if tagged_sql.Error != nil {
		t.Fatal(tagged_sql.Error)
	}

	rows, err := TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
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
			&l.LikeCount,
			&l.CopyCount,
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

func TestTmapTaggedAsSignedInUser(t *testing.T) {
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).AsSignedInUser(TEST_REQ_USER_ID)
	if tagged_sql.Error != nil {
		t.Fatal(tagged_sql.Error)
	}

	rows, err := TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTmapTaggedNSFW(t *testing.T) {
	// TEST_LOGIN_NAME (jlk) has tagged 1 link with NSFW tag
	copied_sql := NewTmapTagged(TEST_LOGIN_NAME).NSFW()
	rows, err := TestClient.Query(copied_sql.Text, copied_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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
		t.Fatal("jlk's tmap does not but should contain 1 tagged link with NSFW tag")
	}
}

func TestTmapTaggedDuringPeriod(t *testing.T) {
	var tagged_no_period, tagged_period_all, tagged_period_week []model.TmapLink
	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME)
	rows, err := TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		}

		tagged_no_period = append(tagged_no_period, l)
	}

	tagged_sql = NewTmapTagged(TEST_LOGIN_NAME).DuringPeriod("all")
	rows, err = TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

	tagged_sql = NewTmapTagged(TEST_LOGIN_NAME).DuringPeriod("week")
	rows, err = TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
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

func TestTmapTaggedWithURLContaining(t *testing.T) {
	url_snippet := "cars" // should be 1 link tagged with URL containing "cars" 
	var expected_count int
	expected_count_sql := `WITH UserCats AS (
		SELECT link_id, cats as user_cats
		FROM user_cats_fts
		WHERE submitted_by = ?
	),
	UserCopies AS (
		SELECT lc.link_id
		FROM "Link Copies" lc
		INNER JOIN Users u ON u.id = lc.user_id
		WHERE u.login_name = ?
	)
	SELECT count(l.id) AS count
	FROM Links l
	INNER JOIN UserCats uct ON l.id = uct.link_id
	LEFT JOIN UserCopies uc ON l.id = uc.link_id
	WHERE l.id NOT IN (
			SELECT link_id FROM global_cats_fts WHERE global_cats MATCH 'NSFW'
	)
	AND l.submitted_by != ?
	AND l.id NOT IN
			(SELECT link_id FROM UserCopies)
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

	tagged_sql := NewTmapTagged(TEST_LOGIN_NAME).WithURLContaining(url_snippet)
	rows, err := TestClient.Query(tagged_sql.Text, tagged_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var links []model.TmapLink
	for rows.Next() {
		link := model.TmapLink{}
		err := rows.Scan(
			&link.ID,
			&link.URL,
			&link.SubmittedBy,
			&link.SubmitDate,
			&link.Cats,
			&link.CatsFromUser,
			&link.Summary,
			&link.SummaryCount,
			&link.LikeCount,
			&link.EarliestLikers,
			&link.CopyCount,
			&link.EarliestCopiers,
			&link.ClickCount,
			&link.TagCount,
			&link.PreviewImgFilename,
		)
		if err != nil {
			t.Fatal(err)
		} 

		links = append(links, link)
	}

	if len(links) != expected_count {
		t.Fatal("len(links) != expected_count")
	}
}

func TestFromUserOrGlobalCats(t *testing.T) {
	tmap_submitted := NewTmapSubmitted(TEST_LOGIN_NAME)
	_, err := TestClient.Query(tmap_submitted.Text, tmap_submitted.Args...)
	if err != nil {
		t.Fatal(err)
	}

	tmap_submitted.Query = FromUserOrGlobalCats(tmap_submitted.Query, test_cats)
	rows, err := TestClient.Query(tmap_submitted.Text, tmap_submitted.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Verify links only have cats from test_cats
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		}
	}

	tmap_copied := NewTmapCopied(TEST_LOGIN_NAME)
	_, err = TestClient.Query(tmap_copied.Text, tmap_copied.Args...)
	if err != nil {
		t.Fatal(err)
	}

	tmap_copied.Query = FromUserOrGlobalCats(tmap_copied.Query, test_cats)
	rows, err = TestClient.Query(tmap_copied.Text, tmap_copied.Args...)
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
			&l.LikeCount,
			&l.EarliestLikers,
			&l.CopyCount,
			&l.EarliestCopiers,
			&l.ClickCount,
			&l.TagCount,
			&l.PreviewImgFilename,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		}
	}

	// TmapTagged does not use FromUserOrGlobalCats()
}
