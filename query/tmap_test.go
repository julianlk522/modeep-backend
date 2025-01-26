package query

import (
	"database/sql"
	"strings"
	"testing"

	"slices"

	"github.com/julianlk522/fitm/model"
)

var (
	test_login_name     = "jlk"
	test_user_id        = "3"
	test_req_login_name = "bradley"
	test_req_user_id    = "13"
	test_cats           = []string{"go", "coding"}
)

func TestNewTmapProfile(t *testing.T) {
	profile_sql := NewTmapProfile(test_login_name)

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
	sql := NewTmapNSFWLinksCount(test_login_name)
	var count int
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// Copied / Tagged
	// test user jlk copied link 76 with global tag "engine,search,NSFW",
	// test user jlk tagged link 9122ce5a-b8ae-4059-afb4-b9ad602c13c2 with
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
	sql = NewTmapNSFWLinksCount(test_req_login_name)
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// Only link test_req_login_name (bradley) has submitted with cat "NSFW" is 76
	// (count should be 1)
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

func TestNewTmapSubmitted(t *testing.T) {
	// Retrieve all IDs of links submitted by user
	var submitted_ids []string
	rows, err := TestClient.Query(`SELECT id 
		FROM Links 
		WHERE submitted_by = ?
		AND global_cats NOT LIKE '%' || 'NSFW' || '%';`, // exclude NSFW in base query
		test_req_login_name)
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
	submitted_sql := NewTmapSubmitted(test_req_login_name)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if l.SubmittedBy != test_req_login_name {
			t.Fatalf("SubmittedBy != test login_name (%s)", test_req_login_name)
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

func TestNewTmapSubmittedFromCats(t *testing.T) {
	submitted_sql := NewTmapSubmitted(test_login_name).FromCats(test_cats)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}
	}
}

func TestNewTmapSubmittedAsSignedInUser(t *testing.T) {
	submitted_sql := NewTmapSubmitted(test_login_name).AsSignedInUser(test_req_user_id)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapSubmittedNSFW(t *testing.T) {
	submitted_sql := NewTmapSubmitted("bradley").NSFW()
	rows, err := TestClient.Query(submitted_sql.Text, submitted_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}

	// Verify test_login_name's tmap contains link with NSFW tag
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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

// Copied
func TestNewTmapCopied(t *testing.T) {
	copied_sql := NewTmapCopied(test_login_name)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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
			test_user_id).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapCopiedFromCats(t *testing.T) {
	copied_sql := NewTmapCopied(test_login_name).FromCats(test_cats)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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
			test_user_id).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapCopiedAsSignedInUser(t *testing.T) {
	copied_sql := NewTmapCopied(test_login_name).AsSignedInUser(test_req_user_id)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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
		WHERE user_id = ?`, test_user_id)
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
	copied_sql = NewTmapCopied(test_login_name).AsSignedInUser(test_req_user_id).NSFW()
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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

func TestNewTmapCopiedNSFW(t *testing.T) {
	// test_login_name (jlk) has copied 1 link with NSFW tag
	copied_sql := NewTmapCopied(test_login_name).NSFW()
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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

func TestNewTmapTagged(t *testing.T) {
	tagged_sql := NewTmapTagged(test_login_name)
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
			&l.ImgURL,
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
			test_login_name).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapTaggedFromCats(t *testing.T) {
	tagged_sql := NewTmapTagged(test_login_name).FromCats(test_cats)
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
			&l.ImgURL,
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
			test_login_name,
		).Scan(&link_id)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapTaggedAsSignedInUser(t *testing.T) {
	tagged_sql := NewTmapTagged(test_login_name).AsSignedInUser(test_req_user_id)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapTaggedNSFW(t *testing.T) {
	// test_login_name (jlk) has tagged 1 link with NSFW tag
	copied_sql := NewTmapTagged(test_login_name).NSFW()
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
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

func TestFromUserOrGlobalCats(t *testing.T) {
	tmap_submitted := NewTmapSubmitted(test_login_name)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		}
	}

	tmap_copied := NewTmapCopied(test_login_name)
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
			&l.CopyCount,
			&l.ClickCount,
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		}
	}

	// TmapTagged does not use FromUserOrGlobalCats()
}
