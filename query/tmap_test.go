package query

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/julianlk522/fitm/model"
)

var (
	test_login_name = "jlk"
	test_user_id    = "3"
	test_cats       = []string{"go", "coding"}

	test_req_user_id    = "13"
	test_req_login_name = "bradley"
)

// Profile
func TestNewTmapProfile(t *testing.T) {
	profile_sql := NewTmapProfile(test_login_name)

	var profile model.Profile
	if err := TestClient.QueryRow(profile_sql.Text, profile_sql.Args...).Scan(
		&profile.LoginName,
		&profile.About,
		&profile.PFP,
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

	// test copied / tagged
	// jlk copied link 76 with global tag "engine,search,NSFW", 
	// jlk tagged link 9122ce5a-b8ae-4059-afb4-b9ad602c13c2 with cat "NSFW"
	// (count should be 2)

	expected_count := 2
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}

	// test .FromCats
	sql = sql.FromCats([]string{"engine", "search"})
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// only link 76 has cats "engine" and "search" in addition to "NSFW"
	// (count should be 1)
	expected_count = 1
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}

	// test submitted
	sql = NewTmapNSFWLinksCount(test_req_login_name)
	if err := TestClient.QueryRow(sql.Text, sql.Args...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	// only link bradley has submitted with cat "NSFW" is 76
	// (count should be 1)
	if count != expected_count {
		t.Fatalf("expected %d, got %d", expected_count, count)
	}
}

// Submitted
func TestNewTmapSubmitted(t *testing.T) {

	// first retrieve all IDs of links submitted by user
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

	// execute query and confirm all submitted links are present
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

		// remove from submitted_ids if returned by query
		for i := 0; i < len(submitted_ids); i++ {
			if l.ID == submitted_ids[i] {
				submitted_ids = append(submitted_ids[0:i], submitted_ids[i+1:]...)
				break
			}
		}
	}

	// if any IDs are left in submitted_ids then they were incorrectly
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

	// just test first row since column counts will be the same
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
			&l.TagCount,
			&l.LikeCount,
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

	// verify test_login_name's tmap contains link with NSFW tag
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
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		} else if strings.Contains(l.Cats, "NSFW") {
			t.Fatal("should not contain NSFW in base query")
		}

		// check that tmap owner has copied
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
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}

		// check that tmap owner has copied
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

	// test first row only since column counts will be the same
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
			&l.TagCount,
			&l.LikeCount,
			&l.ImgURL,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapCopiedNSFW(t *testing.T) {
	// test_login_name (jlk) should have copied 1 link with NSFW tag

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

// Tagged
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
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}

		// check that tmap owner has tagged
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
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		} else if l.TagCount == 0 {
			t.Fatal("TagCount == 0")
		}

		// check that tmap owner has tagged
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

	// test first row only since column counts will be the same
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
			&l.TagCount,
			&l.LikeCount,
			&l.ImgURL,
			&l.IsLiked,
			&l.IsCopied,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTmapTaggedNSFW(t *testing.T) {
	// test_login_name (jlk) should have tagged 1 link with NSFW tag

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

	// submitted
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

	// make sure links only have cats from test_cats
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
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		}
	}

	// copied
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
			&l.TagCount,
			&l.ImgURL,
		); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(l.Cats, test_cats[0]) || !strings.Contains(l.Cats, test_cats[1]) {
			t.Fatalf("got %s, should contain %s", l.Cats, test_cats)
		}
	}
}
