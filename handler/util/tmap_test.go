package handler

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"testing"

	m "github.com/julianlk522/fitm/middleware"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

// Get treausre map
func TestUserExists(t *testing.T) {
	var test_login_names = []struct {
		login_name string
		Exists     bool
	}{
		{"johndoe", false},
		{"janedoe", false},
		{test_login_name, true},
	}

	for _, l := range test_login_names {
		return_true, err := UserExists(l.login_name)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.Exists && !return_true {
			t.Fatalf("expected user %s to exist", l.login_name)
		} else if !l.Exists && return_true {
			t.Fatalf("user %s does not exist", l.login_name)
		}
	}
}

func TestGetTmapForUser(t *testing.T) {
	var test_requests = []struct {
		LoginName               string
		RequestingUserID        string
		CatsParams              string
	}{
		{test_login_name, test_user_id, ""},
		{test_login_name, test_req_user_id, ""},
		{test_login_name, "", ""},
		{test_login_name, test_user_id, "umvc3"},
		{test_login_name, test_req_user_id, "umvc3"},
		{test_login_name, "", "umvc3"},
		{test_login_name, test_user_id, "umvc3,flowers"},
	}

	for _, r := range test_requests {
		req := &http.Request{
			URL: &url.URL{
				RawQuery: url.Values{
					"cats": {r.CatsParams},
				}.Encode(),
			},
		}

		ctx := context.Background()
		jwt_claims := map[string]interface{}{
			"user_id":    r.RequestingUserID,
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		req = req.WithContext(ctx)

		var tmap interface{}
		var err error

		if r.RequestingUserID != "" {
			tmap, err = GetTmapForUser[model.TmapLinkSignedIn](r.LoginName, req)
		} else {
			tmap, err = GetTmapForUser[model.TmapLink](r.LoginName, req)
		}
		if err != nil {
			t.Fatalf(
				`failed test with error: %s for %+v`,
				err,
				r,
			)
		}

		// verify type and filter
		var is_filtered bool
		switch tmap.(type) {
		case model.Tmap[model.TmapLink]:
			is_filtered = false
		case model.Tmap[model.TmapLinkSignedIn]:
			is_filtered = false
		case model.FilteredTmap[model.TmapLink]:
			is_filtered = true
		case model.FilteredTmap[model.TmapLinkSignedIn]:
			is_filtered = true
		}

		if is_filtered && r.CatsParams == "" {
			t.Fatalf("expected unfiltered treasure map type, got %T", tmap)
		} else if !is_filtered && r.CatsParams != "" {
			t.Fatalf("expected filtered treasure map type, got %T", tmap)
		}
	}
}

func TestScanTmapProfile(t *testing.T) {
	profile_sql := query.NewTmapProfile(test_login_name)
	// NewTmapProfile() tested in query/tmap_test.go

	profile, err := ScanTmapProfile(profile_sql)
	if err != nil {
		t.Fatal(err)
	}

	if profile.LoginName != test_login_name {
		t.Fatalf(
			"expected %s, got %s", test_login_name,
			profile.LoginName,
		)
	}

	if profile.Created != "2024-04-10T03:48:09Z" {
		t.Fatalf(
			"expected %s, got %s", "2024-04-10T03:48:09Z",
			profile.Created,
		)
	}
}

func TestScanTmapLinks(t *testing.T) {
	var test_requests = []struct {
		LoginName               string
		RequestingUserID        string
	}{
		{test_login_name, test_user_id},
		{test_login_name, test_req_user_id},
		{test_login_name, ""},
	}

	for _, r := range test_requests {
		submitted_sql := query.NewTmapSubmitted(r.LoginName)
		copied_sql := query.NewTmapCopied(r.LoginName)
		tagged_sql := query.NewTmapTagged(r.LoginName)

		if r.RequestingUserID != "" {
			submitted_sql = submitted_sql.AsSignedInUser(r.RequestingUserID)
			copied_sql = copied_sql.AsSignedInUser(r.RequestingUserID)
			tagged_sql = tagged_sql.AsSignedInUser(r.RequestingUserID)

			_, err := ScanTmapLinks[model.TmapLinkSignedIn](submitted_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap submitted links (signed-in) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLinkSignedIn](copied_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap copied links (signed-in) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLinkSignedIn](tagged_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap tagged links (signed-in) with error: %s",
					err,
				)
			}
		} else {
			_, err := ScanTmapLinks[model.TmapLink](submitted_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap submitted links (no auth) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLink](copied_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap copied links (no auth) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLink](tagged_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap tagged links (no auth) with error: %s",
					err,
				)
			}
		}
	}
}

// Cat counts
func TestGetCatCountsFromTmapLinks(t *testing.T) {
	mock_request := &http.Request{
		URL: &url.URL{
			RawQuery: url.Values{
				"cats": {""},
			}.Encode(),
		},
	}

	ctx := context.Background()
	jwt_claims := map[string]interface{}{
		"user_id":    "",
		"login_name": "",
	}
	ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
	mock_request = mock_request.WithContext(ctx)

	tmap, err := GetTmapForUser[model.TmapLink]("xyz", mock_request)
	if err != nil {
		t.Fatalf("failed with error %s", err)
	}

	var all_links interface{}

	switch tmap.(type) {
	case model.Tmap[model.TmapLink]:
		all_links = slices.Concat(
			*tmap.(model.Tmap[model.TmapLink]).Submitted,
			*tmap.(model.Tmap[model.TmapLink]).Copied,
			*tmap.(model.Tmap[model.TmapLink]).Tagged,
		)
		l, ok := all_links.([]model.TmapLink)
		if !ok {
			t.Fatalf("unexpected type %T", all_links)
		}

		// test without any omitted cats
		var unfiltered_test_cat_counts = []struct {
			Cat   string
			Count int32
		}{
			{"test", 2},
			{"flowers", 1},
		}

		cat_counts := GetCatCountsFromTmapLinks(&l, nil)
		for _, count := range *cat_counts {
			for _, test_count := range unfiltered_test_cat_counts {
				if count.Category == test_count.Cat && count.Count != test_count.Count {
					t.Fatalf(
						"expected count %d for cat %s, got %d",
						test_count.Count,
						test_count.Cat,
						count.Count,
					)
				}
			}
		}

		// test with empty omitted cats
		// (should never happen, but should behave as if no omitted cats were passed)
		cat_counts = GetCatCountsFromTmapLinks(
			&l,
			&model.TmapCatCountsOpts{
				OmittedCats: []string{},
			},
		)

		for _, count := range *cat_counts {
			for _, test_count := range unfiltered_test_cat_counts {
				if count.Category == test_count.Cat && count.Count != test_count.Count {
					t.Fatalf(
						"expected count %d for cat %s, got %d",
						test_count.Count,
						test_count.Cat,
						count.Count,
					)
				}
			}
		}

		// test with omitted cats
		var filtered_test_cat_counts = []struct {
			Cat   string
			Count int32
		}{
			{"test", 0},
			{"flowers", 1},
		}
		var omit = []string{"test"}

		cat_counts = GetCatCountsFromTmapLinks(
			&l,
			&model.TmapCatCountsOpts{
				OmittedCats: omit,
			},
		)
		for _, count := range *cat_counts {
			for _, test_count := range filtered_test_cat_counts {
				if count.Category == test_count.Cat && count.Count != test_count.Count {
					t.Fatalf(
						"expected count %d for cat %s, got %d",
						test_count.Count,
						test_count.Cat,
						count.Count,
					)
				}
			}
		}
	default:
		t.Fatalf("unexpected tmap type %T", tmap)
	}
}
