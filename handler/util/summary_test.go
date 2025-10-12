package handler

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/julianlk522/modeep/db"
	m "github.com/julianlk522/modeep/middleware"
	"github.com/julianlk522/modeep/model"
)

// Get summaries
func TestBuildSummaryPageForLink(t *testing.T) {
	ctx := context.Background()
	jwt_claims := map[string]any{
		"user_id":    TEST_USER_ID,
		"login_name": TEST_LOGIN_NAME,
	}
	ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
	r := (&http.Request{}).WithContext(ctx)

	summary_page, err := BuildSummaryPageForLink(TEST_LINK_ID, r)
	if err != nil {
		t.Fatalf("could not get summary page: %s", err)
	}

	if summary_page, ok := summary_page.(model.SummaryPage[model.SummarySignedIn, model.LinkSignedIn]); ok {

		// Verify summaries are all for provided link
		for _, summary := range summary_page.Summaries {
			var link_id string
			err := TestClient.QueryRow(`
				SELECT link_id 
				FROM Summaries 
				WHERE id = ?`,
				summary.ID).Scan(&link_id)
			if err != nil {
				t.Fatalf(
					"failed to verify summary link: %s (summary ID %s)",
					err,
					summary.ID,
				)
			} else if link_id != TEST_LINK_ID {
				t.Fatalf(
					"summary %s does not belong to link %s",
					summary.ID,
					TEST_LINK_ID,
				)
			}
		}

		// Verify starred count
		var stc int64
		var tc int

		err = TestClient.QueryRow(`
			SELECT count(*)
			FROM Stars
			WHERE link_id = ?`,
			TEST_LINK_ID).Scan(&stc)

		if err != nil {
			t.Fatalf("failed to get link times_starred: %s", err)
		} else if stc != summary_page.Link.TimesStarred {
			t.Fatalf("got link like count %d, want %d", stc, summary_page.Link.TimesStarred)
		}

		// Verify tag count
		err = TestClient.QueryRow(`
			SELECT count(*)
			FROM Tags
			WHERE link_id = ?`,
			TEST_LINK_ID).Scan(&tc)

		if err != nil {
			t.Fatalf("failed to get link tag count: %s", err)
		} else if tc != summary_page.Link.TagCount {
			t.Fatalf("got link tag count %d, want %d", tc, summary_page.Link.TagCount)
		}

		// Verify summary count
		var sc int
		err = TestClient.QueryRow(`
			SELECT count(*)
			FROM Summaries
			WHERE link_id = ?`,
			TEST_LINK_ID).Scan(&sc)
		if err != nil {
			t.Fatalf("failed to get link summary count: %s", err)
		} else if sc != summary_page.Link.SummaryCount {
			t.Fatalf("got link summary count %d, want %d", sc, summary_page.Link.SummaryCount)
		}
	} else {
		t.Fatalf("unexpected summary page shape")
	}
}

// Add summary
func TestLinkExists(t *testing.T) {
	var test_link_ids = []struct {
		ID     string
		Exists bool
	}{
		{"1", true},
		{"2", false},
		{"7", false},
		{"24", true},
		{"87", false},
	}

	for _, l := range test_link_ids {
		got, err := LinkExists(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.Exists != got {
			t.Fatalf("expected %t, got %t for link %s", l.Exists, got, l.ID)
		}
	}
}

func TestGetSummaryIDForLink(t *testing.T) {
	var test_summary_id, test_link_id = "84", "99"

	summary_id, err := GetIDOfUserSummaryForLink(TEST_USER_ID, test_link_id)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	} else if summary_id != test_summary_id {
		t.Fatalf("got summary ID %s, want %s", summary_id, test_summary_id)
	}
}

// Delete summary
func TestGetLinkIDFromSummaryID(t *testing.T) {
	var test_summary_id, test_link_id = "84", "99"

	link_id, err := GetLinkIDFromSummaryID(test_summary_id)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	} else if link_id != test_link_id {
		t.Fatalf("got link ID %s, want %s", link_id, test_link_id)
	}
}

func TestLinkHasOneSummaryLeft(t *testing.T) {
	// var one_summary = "0"
	// var multiple_summaries = "1"
	// var no_summaries = "81"

	var test_link_ids = []struct {
		ID            string
		SingleSummary bool
	}{
		{"0", true},
		{"1", false},
		{"81", false},
	}

	for _, l := range test_link_ids {
		got, err := LinkHasOneSummaryLeft(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.SingleSummary != got {
			t.Fatalf("expected %t, got %t", l.SingleSummary, got)
		}
	}
}

// Like / unlike summary
func TestSummarySubmittedByUser(t *testing.T) {
	var test_summary_ids = []struct {
		ID                  string
		SubmittedByTestUser bool
	}{
		{"7", false},
		{"13", false},
		{"23", false},
		{"65", true},
		{"78", true},
		{"86", false},
	}

	for _, l := range test_summary_ids {
		got, err := SummarySubmittedByUser(l.ID, TEST_USER_ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.SubmittedByTestUser != got {
			t.Fatalf("expected %t, got %t for link %s", l.SubmittedByTestUser, got, l.ID)
		}
	}
}

func TestIsAutoSummaryForLinkSubmittedByUser(t *testing.T) {
	all_auto_summaries_sql := `SELECT id, link_id
	FROM Summaries
	WHERE submitted_by = ?`
	rows, err := db.Client.Query(all_auto_summaries_sql, db.AUTO_SUMMARY_USER_ID)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	}
	defer rows.Close()

	var auto_summaries []struct {
		ID     string
		LinkID string
	}
	for rows.Next() {
		var auto_summary struct {
			ID     string
			LinkID string
		}
		if err := rows.Scan(
			&auto_summary.ID,
			&auto_summary.LinkID,
		); err != nil {
			t.Fatalf("failed with error: %s", err)
		}

		auto_summaries = append(auto_summaries, auto_summary)
	}

	test_user_links_sql := `SELECT id
	FROM Links
	WHERE submitted_by IN (
		SELECT login_name
		FROM Users
		WHERE id = ?
	)`
	rows, err = db.Client.Query(
		test_user_links_sql,
		TEST_USER_ID,
	)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	}
	defer rows.Close()
	
	var test_user_link_ids []string
	for rows.Next() {
		var test_user_link_id string
		if err := rows.Scan(&test_user_link_id); err != nil {
			t.Fatalf("failed with error: %s", err)
		}
		test_user_link_ids = append(test_user_link_ids, test_user_link_id)
	}

	for _, as := range auto_summaries {
		got, err := IsAutoSummaryForLinkSubmittedByUser(
			as.ID,
			TEST_USER_ID,
		)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		}

		// should be in the found link IDs
		if got {
			found := slices.Contains(test_user_link_ids, as.LinkID)

			if !found {
				t.Fatalf("expected %t, got %t", true, got)
			}
		}
	}
}

func TestUserHasLikedSummary(t *testing.T) {
	var test_summary_ids = []struct {
		ID              string
		LikedByTestUser bool
	}{
		{"1", true},
		{"3", false},
		{"4", false},
		{"118", true},
		{"88", true},
		{"35", false},
	}

	for _, l := range test_summary_ids {
		got, err := UserHasLikedSummary(TEST_USER_ID, l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.LikedByTestUser != got {
			t.Fatalf("expected %t, got %t", l.LikedByTestUser, got)
		}
	}
}

// Calculate global summary
func TestCalculateAndSetGlobalSummary(t *testing.T) {
	var test_link_ids = []struct {
		ID            string
		GlobalSummary string
	}{
		{"10", "Doesn't seem to be a real site..."},
		{"93", "The very first website!"},
	}

	for _, l := range test_link_ids {
		err := CalculateAndSetGlobalSummary(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		}

		// confirm global summary matches expected
		var gs string
		err = TestClient.QueryRow(`
			SELECT global_summary 
			FROM Links 
			WHERE id = ?`,
			l.ID,
		).Scan(&gs)

		if err != nil {
			t.Fatalf(
				"failed with error: %s for link with ID %s",
				err,
				l.ID,
			)
		} else if gs != l.GlobalSummary {
			t.Fatalf(
				"got global summary %s for link with ID %s, want %s",
				gs,
				l.ID,
				l.GlobalSummary,
			)
		}
	}
}
