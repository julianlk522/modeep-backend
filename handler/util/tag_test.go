package handler

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/julianlk522/modeep/query"
)

func TestGetUserTagForLink(t *testing.T) {
	var test_tag = struct {
		LoginName string
		LinkID    string
		Cats      string
	}{
		LoginName: TEST_LOGIN_NAME,
		LinkID:    "22",
		Cats:      "barbie,magic,wow",
	}

	tag, err := GetUserTagForLink(test_tag.LoginName, test_tag.LinkID)
	if err != nil {
		t.Fatal(err)
	} else if tag == nil {
		t.Fatalf(
			"no tag found for user %s and link %s, expected cats %s",
			test_tag.LoginName,
			test_tag.LinkID,
			test_tag.Cats,
		)
	}

	// Verify id and cats
	var id, cats string

	err = TestClient.QueryRow(`
		SELECT id, cats 
		FROM Tags 
		WHERE submitted_by = ?
		AND link_id = ?;`,
		test_tag.LoginName,
		test_tag.LinkID).Scan(
		&id,
		&cats,
	)
	if err != nil {
		t.Fatal(err)
	} else if tag.ID != id {
		t.Fatalf(
			"got tag ID %s for user %s and link %s, want %s",
			tag.ID,
			test_tag.LoginName,
			test_tag.LinkID,
			id,
		)
	} else if tag.Cats != cats {
		t.Fatalf(
			"got cats %s for user %s and link %s, want %s",
			tag.Cats,
			test_tag.LoginName,
			test_tag.LinkID,
			cats,
		)
	}
}

func TestScanTagRankings(t *testing.T) {
	tag_rankings_sql := query.NewTagRankingsForLink(TEST_LINK_ID)
	if tag_rankings_sql.Error != nil {
		t.Fatal(tag_rankings_sql.Error)
	}

	if _, err := ScanTagRankings(tag_rankings_sql); err != nil {
		t.Fatal(err)
	}
}

func TestScanGlobalCatCounts(t *testing.T) {
	global_cats_sql := query.NewTopGlobalCatCounts()
	if _, err := ScanGlobalCatCounts(global_cats_sql); err != nil {
		t.Fatal(err)
	}
}

func TestCatsResembleEachOther(t *testing.T) {
	var test_cats = []struct {
		CatA           string
		CatB           string
		ExpectedResult bool
	}{
		{"test", "tests", true},
		{"dresses", "dress", true},
		{"game", "games", true},
		{"glitch", "glitches", true},
		{"test", "abc", false},
		// still valid if same spelling but different case
		{"abc", "abc", true},
		{"abc", "ABC", true},
	}

	for _, c := range test_cats {
		got := CatsResembleEachOther(c.CatA, c.CatB)
		if c.ExpectedResult != got {
			t.Fatalf(
				"expected %t, got %t for cats %s and %s",
				c.ExpectedResult,
				got,
				c.CatA,
				c.CatB,
			)
		}
	}
}

func TestTidyCats(t *testing.T) {
	var test_cats = []struct {
		Cats           string
		ExpectedResult string
	}{
		{"", ""},
		{"test,abc", "abc,test"},
		{"  test,abc  ", "abc,test"},
		{"abc,test,ACB", "abc,ACB,test"},
	}

	for _, c := range test_cats {
		got := TidyCats(c.Cats)
		if c.ExpectedResult != got {
			t.Fatalf("expected %s, got %s", c.ExpectedResult, got)
		}
	}
}

func TestUserHasTaggedLink(t *testing.T) {
	var test_links = []struct {
		ID               string
		TaggedByTestUser bool
	}{
		{"1", true},
		{"102", true},
		{"22", true},
		{"13", false},
		{"0", false},
	}

	for _, l := range test_links {
		got, err := UserHasTaggedLink(TEST_LOGIN_NAME, l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.TaggedByTestUser != got {
			t.Fatalf("expected %t, got %t for link %s", l.TaggedByTestUser, got, l.ID)
		}
	}
}

func TestUserSubmittedTagWithID(t *testing.T) {
	var test_tags = []struct {
		ID                  string
		SubmittedByTestUser bool
	}{
		{"32", true},
		{"34", false},
		{"114", true},
		{"5", false},
		{"6", false},
	}

	for _, tag := range test_tags {
		got, err := UserSubmittedTagWithID(TEST_LOGIN_NAME, tag.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if tag.SubmittedByTestUser != got {
			t.Fatalf("expected %t, got %t for tag %s", tag.SubmittedByTestUser, got, tag.ID)
		}
	}
}

func TestTagExists(t *testing.T) {
	// TODO
}

func TestIsOnlyTag(t *testing.T) {
	var test_tags = []struct {
		ID     string
		IsOnly bool
	}{
		{"5", true},
		{"4", false},
		{"35", true},
	}

	for _, tag := range test_tags {
		got, err := IsOnlyTag(tag.ID)
		if err != nil {
			t.Fatalf("failed with error: %s for tag %s", err, tag.ID)
		} else if tag.IsOnly != got {
			t.Fatalf("expected %t, got %t for tag %s", tag.IsOnly, got, tag.ID)
		}
	}
}

func TestGetLinkIDFromTagID(t *testing.T) {
	var test_tags = []struct {
		ID     string
		LinkID string
	}{
		{"32", "1"},
		{"114", "22"},
		{"5", "0"},
		{"6", "8"},
		{"11", "10"},
	}

	for _, tag := range test_tags {
		return_link_id, err := GetLinkIDFromTagID(tag.ID)
		if err != nil {
			t.Fatalf("failed with error: %s for tag %s", err, tag.ID)
		} else if tag.LinkID != return_link_id {
			t.Fatalf(
				"expected tag with ID %s to have link ID %s",
				tag.ID,
				tag.LinkID,
			)
		}
	}
}

func TestCalculateAndSetGlobalCats(t *testing.T) {
	var test_link_ids = []struct {
		ID                 string
		ExpectedGlobalCats string
	}{
		{"0", "flowers"},
		{"11", "test"},
		{"1234567890", "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20"},
	}

	for _, l := range test_link_ids {
		err := CalculateAndSetGlobalCats(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s for link with ID %s", err, l.ID)
		}

		// confirm global cats match expected
		var gc string
		err = TestClient.QueryRow(`
			SELECT global_cats
			FROM Links 
			WHERE id = ?`,
			l.ID,
		).Scan(&gc)

		if err != nil {
			t.Fatalf(
				"failed with error: %s for link with ID %s",
				err,
				l.ID,
			)
		} else if gc != l.ExpectedGlobalCats {
			t.Fatalf(
				"got global cats %s for link with ID %s, want %s",
				gc,
				l.ID,
				l.ExpectedGlobalCats,
			)
		}
	}
}

func TestSetGlobalCats(t *testing.T) {
	var test_link_id = "11"
	var test_cats = "example,cats"

	// get old spellfix ranks for test cats
	var old_test_cats_ranks = make(map[string]int)
	for cat := range strings.SplitSeq(test_cats, ",") {
		var rank int
		err := TestClient.QueryRow(`
			SELECT rank
			FROM global_cats_spellfix
			WHERE word = ?`,
			cat,
		).Scan(&rank)
		if err != nil {
			if err != sql.ErrNoRows {
				t.Fatalf("failed with error: %s", err)
			}
			old_test_cats_ranks[cat] = 0
		}
		old_test_cats_ranks[cat] = rank
	}

	// get old spellfix ranks for link's global cats
	var old_link_gcs string
	err := TestClient.QueryRow(`
		SELECT global_cats
		FROM Links 
		WHERE id = ?`,
		test_link_id,
	).Scan(&old_link_gcs)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	}

	var old_link_gc_ranks = make(map[string]int)
	for cat := range strings.SplitSeq(old_link_gcs, ",") {
		var rank int
		err := TestClient.QueryRow(`
			SELECT rank
			FROM global_cats_spellfix
			WHERE word = ?`,
			cat,
		).Scan(&rank)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		}
		old_link_gc_ranks[cat] = rank
	}

	err = setGlobalCats(test_link_id, test_cats)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	}

	// verify global cats match expected
	var new_link_gc string
	err = TestClient.QueryRow(`
		SELECT global_cats
		FROM Links 
		WHERE id = ?`,
		test_link_id,
	).Scan(&new_link_gc)

	if err != nil {
		t.Fatalf("failed with error: %s", err)
	} else if new_link_gc != test_cats {
		t.Fatalf("got global cats %s, want %s", new_link_gc, test_cats)
	}

	// verify test cats spellfix ranks have been incremented
	for cat, old_rank := range old_test_cats_ranks {
		var new_rank int
		err := TestClient.QueryRow(`
			SELECT rank
			FROM global_cats_spellfix
			WHERE word = ?`,
			cat,
		).Scan(&new_rank)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		}
		if new_rank != old_rank+1 {
			t.Fatalf(
				"expected rank for %s to be %d, got %d",
				cat,
				old_rank+1,
				new_rank,
			)
		}
	}

	// verify old global cats spellfix ranks have been decremented
	for cat, old_rank := range old_link_gc_ranks {
		var new_rank int
		err := TestClient.QueryRow(`
			SELECT rank
			FROM global_cats_spellfix
			WHERE word = ?`,
			cat,
		).Scan(&new_rank)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		}
		if new_rank != old_rank-1 {
			t.Fatalf(
				"expected rank for %s to be %d, got %d",
				cat,
				old_rank-1,
				new_rank,
			)
		}
	}
}
