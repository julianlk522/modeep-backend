package handler

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

func TestScanTagPageLink(t *testing.T) {
	link_sql := query.NewTagPageLink(test_link_id)
	// NewTagPageLink().Error already tested in query/tag_test.go

	// signed out
	link, err := ScanTagPageLink[model.Link](link_sql)
	if err != nil {
		t.Fatal(err)
	} else if link == nil {
		t.Fatal("no link (signed out)")
	}

	// signed in
	link_sql = link_sql.AsSignedInUser(test_req_user_id)
	link2, err := ScanTagPageLink[model.LinkSignedIn](link_sql)
	if err != nil {
		t.Fatal(err)
	} else if link2 == nil {
		t.Fatal("no link (signed in)")
	}

	// Verify link ID
	if link2.ID != test_link_id {
		t.Fatalf(
			"got link ID %s, want %s",
			link2.ID,
			test_link_id,
		)
	}

	// Verify isLiked / isCopied
	liked := UserHasLikedLink(test_req_user_id, test_link_id)
	if liked && !link2.IsLiked {
		t.Fatalf("expected link with ID %s to be liked by user", test_link_id)
	} else if !liked && link2.IsLiked {
		t.Fatalf("link with ID %s NOT liked by user, expected error", test_link_id)
	}

	copied := UserHasCopiedLink(test_req_user_id, test_link_id)
	if copied && !link2.IsCopied {
		t.Fatalf("expected link with ID %s to be copied by user", test_link_id)
	} else if !copied && link2.IsCopied {
		t.Fatalf("link with ID %s NOT copied by user, expected error", test_link_id)
	}
}

func TestGetUserTagForLink(t *testing.T) {
	var test_tag = struct {
		LoginName string
		LinkID    string
		Cats      string
	}{
		LoginName: test_login_name,
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
	var test_rankings = []struct {
		Cats        string
		SubmittedBy string
	}{
		{
			Cats:        "flowers",
			SubmittedBy: "xyz",
		},
		{
			Cats:        "jungle,idk,something",
			SubmittedBy: "nelson",
		},
		{
			Cats:        "i,hate,sql",
			SubmittedBy: "Test User",
		},
		{
			Cats:        "monkeys,something",
			SubmittedBy: "jlk",
		},
		{
			Cats:        "jungle,knights,monkeys,talladega",
			SubmittedBy: "monkey",
		},
	}
	tag_rankings_sql := query.NewTagRankings(test_link_id).Public()
	// NewTagRankings(link_id).Public().Error already tested in query/tag_test.go

	rankings, err := ScanPublicTagRankings(tag_rankings_sql)
	if err != nil {
		t.Fatal(err)
	}

	// Verify result length
	if len(*rankings) != len(test_rankings) {
		t.Fatalf(
			"got %d tag rankings, want %d",
			len(*rankings),
			len(test_rankings),
		)
	}

	// Verify result order
	for i, ranking := range *rankings {
		if ranking.SubmittedBy != test_rankings[i].SubmittedBy {
			t.Fatalf(
				"expected ranking %d to be submitted by %s, got %s",
				i+1,
				test_rankings[i].SubmittedBy,
				ranking.SubmittedBy,
			)
		} else if ranking.Cats != test_rankings[i].Cats {
			t.Fatalf(
				"expected ranking %d to have cats %s, got %s",
				i+1,
				test_rankings[i].Cats,
				ranking.Cats,
			)
		}
	}
}

// Get top global cats
// (and subcats of cats)
func TestScanGlobalCatCounts(t *testing.T) {
	global_cats_sql := query.NewTopGlobalCatCounts()
	// GlobalCatCounts.Error already tested in query/tag_test.go

	counts, err := ScanGlobalCatCounts(global_cats_sql)
	if err != nil {
		t.Fatal(err)
	}

	if len(*counts) == 0 {
		t.Fatal("no counts returned for top global cats")
	} else if len(*counts) > query.GLOBAL_CATS_PAGE_LIMIT {
		t.Fatalf(
			"too many counts returned for top global cats (limit %d)",
			query.GLOBAL_CATS_PAGE_LIMIT,
		)
	}

	// Verify count for top few cats
	const FEW = 3
	if len(*counts) > FEW {
		*counts = (*counts)[0:FEW]
	}

	var result_count int32

	for _, c := range *counts {
		if c.Count == 0 {
			t.Fatalf("cat %s returned count 0", c.Category)
		}

		err = TestClient.QueryRow(`SELECT count(global_cats)
				FROM Links
				WHERE ','||global_cats||',' LIKE '%,' || ? || ',%'`,
				c.Category).Scan(&result_count)

		if err != nil {
			t.Fatal(err)
		} else if c.Count != result_count {
			t.Fatalf(
				"expected count for cat %s to be %d, got %d",
				c.Category,
				c.Count,
				result_count,
			)
		}
	}

	// DURING PERIOD
	var test_periods = []struct {
		Period string
		Valid  bool
	}{
		{"day", true},
		{"week", true},
		{"month", true},
		{"year", true},
		{"all", false},
		{"invalid_period", false},
	}

	for _, tp := range test_periods {
		global_cats_sql = query.NewTopGlobalCatCounts().DuringPeriod(tp.Period)
		// GlobalCatCounts.DuringPeriod().Error already tested
		// in query/tag_test.go with same test cases

		counts, err := ScanGlobalCatCounts(global_cats_sql)
		if tp.Valid && err != nil && err != sql.ErrNoRows {
			t.Fatalf(
				"unexpected error for period %s: %s",
				tp.Period,
				err,
			)
		} else if !tp.Valid && err == nil {
			t.Fatalf("expected error for period %s", tp.Period)
		}

		// Verify counts if valid sql
		if !tp.Valid {
			continue
		}

		if len(*counts) > query.GLOBAL_CATS_PAGE_LIMIT {
			t.Fatalf(
				"too many counts returned for top global cats (limit %d)",
				query.GLOBAL_CATS_PAGE_LIMIT,
			)

			// Only top few cats
		} else if len(*counts) > FEW {
			*counts = (*counts)[0:FEW]
		}

		for _, c := range *counts {
			if c.Count == 0 {
				t.Fatalf("cat %s returned count 0", c.Category)
			}

			period_clause, err := query.GetPeriodClause(tp.Period)
			if err != nil {
				t.Fatalf("unable to get period clause: %s", err)
			}

			err = TestClient.QueryRow(`SELECT count(global_cats)
					FROM Links
					WHERE ',' || global_cats || ',' LIKE '%,' || ? || ',%'
					AND ` + period_clause + ";",
					c.Category,
					period_clause).Scan(&result_count)

			if err != nil {
				t.Fatal(err)
			} else if c.Count != result_count {
				t.Fatalf(
					"expected count for cat %s to be %d, got %d (period %s)",
					c.Category,
					c.Count,
					result_count,
					tp.Period,
				)
			}
		}
	}
}

func TestUserHasTaggedLink(t *testing.T) {
	var test_links = []struct {
		ID               string
		TaggedByTestUser bool
	}{
		{"1", true},
		{"13", true},
		{"22", true},
		{"0", false},
		{"10", false},
		{"102", false},
	}

	for _, l := range test_links {
		return_true, err := UserHasTaggedLink(test_login_name, l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.TaggedByTestUser && !return_true {
			t.Fatalf("expected tag with ID %s to be tagged by user", l.ID)
		} else if !l.TaggedByTestUser && return_true {
			t.Fatalf("tag with ID %s NOT submitted by user, expected error", l.ID)
		}
	}
}

// Edit tag
func TestUserSubmittedTagWithID(t *testing.T) {
	var test_tags = []struct {
		ID                  string
		SubmittedByTestUser bool
	}{
		{"32", true},
		{"34", true},
		{"114", true},
		{"5", false},
		{"6", false},
		{"11", false},
	}

	for _, tag := range test_tags {
		return_true, err := UserSubmittedTagWithID(test_login_name, tag.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if tag.SubmittedByTestUser && !return_true {
			t.Fatalf("expected tag with ID %s to be submitted by user", tag.ID)
		} else if !tag.SubmittedByTestUser && return_true {
			t.Fatalf("tag with ID %s NOT submitted by user, expected error", tag.ID)
		}
	}
}

// Delete tag
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
		{"34", false},
	}

	for _, tag := range test_tags {
		return_true, err := IsOnlyTag(tag.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if tag.IsOnly != return_true {
			t.Fatalf("expected tag with ID %s to be only tag", tag.ID)
		}
	}
}

// AlphabetizeCats() is simple usage of strings.Split / string.Join / slices.Sort
// no point in testing

func TestGetLinkIDFromTagID(t *testing.T) {
	var test_tags = []struct {
		ID     string
		LinkID string
	}{
		{"32", "1"},
		{"34", "13"},
		{"114", "22"},
		{"5", "0"},
		{"6", "8"},
		{"11", "10"},
	}

	for _, tag := range test_tags {
		return_link_id, err := GetLinkIDFromTagID(tag.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
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
		ID         string
		GlobalCats string
	}{
		{"0", "flowers"},
		{"7", "7,lucky,arrest,Best,jest,Molest,winchest"},
		{"11", "test"},
	}

	for _, l := range test_link_ids {
		err := CalculateAndSetGlobalCats(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
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
		} else if gc != l.GlobalCats {
			t.Fatalf(
				"got global cats %s for link with ID %s, want %s",
				gc,
				l.ID,
				l.GlobalCats,
			)
		}
	}
}

// AlphabetizeOverlapScoreCats() is simple usage of slices.Sort()
// no point in testing

func TestSetGlobalCats(t *testing.T) {
	var test_link_id = "11"
	var test_cats = "reference,food"

	// get old spellfix ranks for test cats
	var old_test_cats_ranks = make(map[string]int)
	for _, cat := range strings.Split(test_cats, ",") {
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
	for _, cat := range strings.Split(old_link_gcs, ",") {
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

	err = SetGlobalCats(test_link_id, test_cats)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	}

	// confirm global cats match expected
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

	// confirm test cats spellfix ranks have been incremented
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

	// confirm old global cats spellfix ranks have been decremented
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
