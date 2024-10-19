package handler

import (
	"context"
	"net/http"
	"testing"

	m "github.com/julianlk522/fitm/middleware"
	"github.com/julianlk522/fitm/model"
)

// Get summaries
func TestBuildSummaryPageForLink(t *testing.T) {
	ctx := context.Background()
	jwt_claims := map[string]interface{}{
		"user_id":    test_user_id,
		"login_name": test_login_name,
	}
	ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
	r := (&http.Request{}).WithContext(ctx)

	summary_page, err := BuildSummaryPageForLink(test_link_id, r)
	if err != nil {
		t.Fatalf("could not get summary page: %s", err)
	}

	if summary_page, ok := summary_page.(model.SummaryPage[model.SummarySignedIn, model.LinkSignedIn]); ok {

		// Check that summaries are all for link provided
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
			} else if link_id != test_link_id {
				t.Fatalf(
					"summary %s does not belong to link %s",
					summary.ID,
					test_link_id,
				)
			}
		}

		// Verify like count
		var lc int64
		var tc int

		err = TestClient.QueryRow(`
			SELECT count(*)
			FROM "Link Likes"
			WHERE link_id = ?`,
			test_link_id).Scan(&lc)

		if err != nil {
			t.Fatalf("failed to get link LikeCount: %s", err)
		} else if lc != summary_page.Link.LikeCount {
			t.Fatalf("got link like count %d, want %d", lc, summary_page.Link.LikeCount)
		}

		// Verify tag count
		err = TestClient.QueryRow(`
			SELECT count(*)
			FROM Tags
			WHERE link_id = ?`,
			test_link_id).Scan(&tc)

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
			test_link_id).Scan(&sc)
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
		{"3", false},
		{"7", true},
		{"24", true},
		{"87", false},
	}

	for _, l := range test_link_ids {
		return_true, err := LinkExists(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.Exists && !return_true {
			t.Fatalf("expected link with ID %s to exist", l.ID)
		} else if !l.Exists && return_true {
			t.Fatalf("link with ID %s does not exist", l.ID)
		}
	}
}

func TestGetSummaryIDForLink(t *testing.T) {
	var test_summary_id, test_link_id = "84", "99"

	summary_id, err := GetIDOfUserSummaryForLink(test_user_id, test_link_id)
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
		return_true, err := LinkHasOneSummaryLeft(l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.SingleSummary && !return_true {
			t.Fatalf("expected link with ID %s to have one summary left", l.ID)
		} else if !l.SingleSummary && return_true {
			t.Fatalf("link with ID %s does NOT have one summary left", l.ID)
		}
	}
}

// Like / unlike summary
func TestSummarySubmittedByUser(t *testing.T) {
	var test_summary_ids = []struct {
		ID                  string
		SubmittedByTestUser bool
	}{
		{"7", true},
		{"13", false},
		{"23", false},
		{"65", true},
		{"78", true},
		{"86", false},
	}

	for _, l := range test_summary_ids {
		return_true, err := SummarySubmittedByUser(l.ID, test_user_id)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.SubmittedByTestUser && !return_true {
			t.Fatalf("expected summary with ID %s to be submitted by user", l.ID)
		} else if !l.SubmittedByTestUser && return_true {
			t.Fatalf("summary with ID %s NOT submitted by user, expected error", l.ID)
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
		return_true, err := UserHasLikedSummary(test_user_id, l.ID)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.LikedByTestUser && !return_true {
			t.Fatalf("expected summary with ID %s to be liked by user", l.ID)
		} else if !l.LikedByTestUser && return_true {
			t.Fatalf("summary with ID %s NOT liked by user, expected error", l.ID)
		}
	}
}

// Calculate global summary
func TestCalculateAndSetGlobalSummary(t *testing.T) {
	var test_link_ids = []struct {
		ID            string
		GlobalSummary string
	}{
		{"1", "test"},
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
