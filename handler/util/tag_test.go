package handler

import (
	"database/sql"
	"strings"
	"testing"

	modelutil "github.com/julianlk522/modeep/model/util"
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

func TestScanPublicTagRankings(t *testing.T) {
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
			Cats:        "jungle,knights,monkeys,talladega",
			SubmittedBy: "monkey",
		},
		{
			Cats:        "hello,kitty",
			SubmittedBy: "jlk",
		},
	}
	tag_rankings_sql := query.NewTagRankings(TEST_LINK_ID).Public()
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
		{"all", true},
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

			counts_sql := `SELECT count(global_cats)
				FROM Links
				WHERE ',' || global_cats || ',' LIKE '%,' || ? || ',%'`

			if tp.Period != "all" {
				period_clause, err := query.GetPeriodClause(tp.Period)
				if err != nil {
					t.Fatalf("unable to get period clause: %s", err)
				}

				counts_sql += "AND "+period_clause+";"
			}

			err = TestClient.QueryRow(counts_sql, c.Category).Scan(&result_count)
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

// Edit tag
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
		// test that calculated global cats are limited to top TAG_CATS_LIMIT
		// link 1234567890 has 1 tag with cats "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17" so should be shortened to top 15
		// (all same scores so sort alphabetically)
		{"1234567890", "1,10,11,12,13,14,15,16,17,2,3,4,5,6,7"},
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

func TestLimitToTopCatRankings(t *testing.T) {
	test_rankings := map[string]float32{
		"cat1":  1,
		"cat2":  2,
		"cat3":  3,
		"cat4":  4,
		"cat5":  5,
		"cat6":  6,
		"cat7":  7,
		"cat8":  8,
		"cat9":  9,
		"cat10": 10,
		"cat11": 11,
		"cat12": 12,
		"cat13": 13,
		"cat14": 14,
		"cat15": 15,
		"cat16": 16,
		"cat17": 17,
	}

	limited_rankings := LimitToTopCatRankings(test_rankings)
	if len(limited_rankings) != modelutil.NUM_CATS_LIMIT {
		t.Fatalf(
			"expected %d cats, got %d",
			modelutil.NUM_CATS_LIMIT,
			len(limited_rankings),
		)
	}

	// test with fewer than TAG_CATS_LIMIT cats just in case, even though
	// this condition should be unreachable
	test_rankings = map[string]float32{
		"cat1": 1,
		"cat2": 2,
	}

	limited_rankings = LimitToTopCatRankings(test_rankings)
	if len(limited_rankings) != len(test_rankings) {
		t.Fatalf(
			"expected %d cats, got %d",
			len(test_rankings),
			len(limited_rankings),
		)
	}
}

func TestSetGlobalCats(t *testing.T) {
	var test_link_id = "11"
	var test_cats = "example,cats"

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
