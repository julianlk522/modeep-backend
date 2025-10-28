package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/julianlk522/modeep/db"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func TestPrepareLinksPage(t *testing.T) {
	var test_requests = []struct {
		LinksOptions   *model.TopLinksOptions
		PageOptions     *model.LinksPageOptions
		Valid      bool
	}{
		{
			LinksOptions:   &model.TopLinksOptions{
				Page: 1,
			},
			PageOptions: &model.LinksPageOptions{},
			Valid:      true,
		},
		{
			LinksOptions: &model.TopLinksOptions{
				CatFiltersWithSpellingVariants: test_multiple_cats,
				Page: 1,
			},
			PageOptions: &model.LinksPageOptions{
				CatFilters: test_multiple_cats, 
			},
			Valid:      true,
		},
		{
			LinksOptions: &model.TopLinksOptions{
				Period: "batman",
				Page: 1,
			},
			Valid:      false,
		},
	}

	for _, tr := range test_requests {
		links_sql, err := query.NewTopLinks().FromOptions(tr.LinksOptions)
		if tr.Valid && err != nil {
			t.Fatal(err)
		} else if !tr.Valid && err == nil {
			t.Fatalf("expected error for request %v", tr)
		}
		
		if !tr.Valid {
			continue
		}

		if _, err = PrepareLinksPage[model.Link](links_sql, tr.PageOptions); err != nil {
			t.Fatalf(
				"got error %s, SQL text was %s, args were %v",
				err,
				links_sql.Text,
				links_sql.Args,
			)
		}
	}
}

func TestScanRawLinksPageData(t *testing.T) {
	links_sql := query.NewTopLinks()

	// signed out
	links_page, err := scanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links_page.Links) == 0 {
		t.Fatal("no links")
	}

	// signed in
	links_sql, err = links_sql.FromOptions(
		&model.TopLinksOptions{
			AsSignedInUser: TEST_REQ_USER_ID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	signed_in_links_page, err := scanRawLinksPageData[model.LinkSignedIn](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*signed_in_links_page.Links) == 0 {
		t.Fatal("no links")
	}
}

func TestScanSingleLink(t *testing.T) {
	// signed out
	single_link_sql := query.NewSingleLink("1")
	if _, err := single_link_sql.ValidateAndExecuteRow(); err != nil {
		t.Fatal(err)
	}

	// signed in
	single_link_sql = single_link_sql.AsSignedInUser(TEST_REQ_USER_ID)
	if _, err := single_link_sql.ValidateAndExecuteRow(); err != nil {
		t.Fatal(err)
	}
}

func TestPaginateLinks(t *testing.T) {
	// single page
	opts := &model.TopLinksOptions{
		CatFiltersWithSpellingVariants: []string{"test"},
		Page: 1,
	}
	links_sql, err := query.NewTopLinks().FromOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	links_page, err := scanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	paginateLinks(links_page.Links)
	if len(*links_page.Links) == 0 {
		t.Fatal("expected links")
	}
}

func TestGetMergedCatSpellingVariantsInLinksFromCatFilters(t *testing.T) {
	// NOTE cats need to have spelling variants added for this to work
	// (done by GetTopLinksOptionsFromRequestParams())

	// no links; no merged cats
	test_cat_filters := []string{"nonexistentcat"}
	opts := &model.TopLinksOptions{
		CatFiltersWithSpellingVariants: query.GetCatsOptionalPluralOrSingularForms(test_cat_filters),
	}
	links_sql, err := query.NewTopLinks().FromOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	links_page, err := scanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}
	links_page.MergedCats = getMergedCatSpellingVariantsInLinksFromCatFilters(
		links_page.Links,
		test_cat_filters,
	)
	if len(links_page.MergedCats) != 0 {
		t.Fatal("expected no merged cats")
	}

	// should merge results for "flowers"
	test_cat_filters = []string{"flower"}
	links_sql, err = query.NewTopLinks().FromOptions(&model.TopLinksOptions{
		CatFiltersWithSpellingVariants: query.GetCatsOptionalPluralOrSingularForms(test_cat_filters),
	})
	if err != nil {
		t.Fatal(err)
	}
	links_page, err = scanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}
	links_page.MergedCats = getMergedCatSpellingVariantsInLinksFromCatFilters(
		links_page.Links,
		test_cat_filters,
	)
	if len(links_page.MergedCats) != 1 {
		t.Fatalf(
			"expected 1 merged cat, got %d (%v)",
			len(links_page.MergedCats),
			links_page.MergedCats,
		)
	}

	// multiple merged cats
	test_cats := []string{"flower", "tests"} // should merge "flowers" and "test"
	links_sql, err = query.NewTopLinks().FromOptions(
		&model.TopLinksOptions{
			CatFiltersWithSpellingVariants: query.GetCatsOptionalPluralOrSingularForms(
				test_cat_filters,
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	links_page, err = scanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}
	links_page.MergedCats = getMergedCatSpellingVariantsInLinksFromCatFilters(
		links_page.Links,
		test_cats,
	)
	if len(links_page.MergedCats) != 2 {
		t.Fatalf(
			"expected 2 merged cats, got %d (%v)",
			len(links_page.MergedCats),
			links_page.MergedCats,
		)
	}

	// inconsistent capitalization: should still merge
	test_cat_filters = []string{"FlOwEr"} // should merge "flowers"
	links_sql, err = query.NewTopLinks().FromOptions(
		&model.TopLinksOptions{
			CatFiltersWithSpellingVariants: query.GetCatsOptionalPluralOrSingularForms(
				test_cat_filters,
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	links_page, err = scanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}
	links_page.MergedCats = getMergedCatSpellingVariantsInLinksFromCatFilters(
		links_page.Links,
		test_cat_filters,
	)
	if len(links_page.MergedCats) != 1 {
		t.Fatalf(
			"expected 1 merged cat, got %d (%v)",
			len(links_page.MergedCats),
			links_page.MergedCats,
		)
	}

}

// Add link
func TestGetLinkExtraMetadataFromResponse(t *testing.T) {
	var test_links = []struct {
		new_link *model.NewLink
		Valid    bool
	}{
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "modeep.org"}}, true},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "www.modeep.org"}}, true},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "https://www.modeep.org"}}, true},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "about.google.com"}}, true},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "julianlk.com/notreal"}}, false},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "gobblety gook"}}, false},
	}

	for _, tl := range test_links {
		req, err := http.NewRequest("GET", tl.new_link.URL, nil)
		if tl.Valid && err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if !tl.Valid && err == nil {
			t.Fatalf("expected error for url %s", tl.new_link.URL)
		}

		x_md := GetLinkExtraMetadataFromResponse(resp)
		if x_md == nil && err == nil {
			t.Fatalf("expected metadata for url %s", tl.new_link.URL)
		}
	}
}

func TestGetResolvedURLResponse(t *testing.T) {
	var test_urls = []struct {
		URL   string
		Valid bool
	}{
		// not having protocol or www subdomain should still work
		{"modeep.org", true},
		{"www.modeep.org", true},
		{"https://www.modeep.org", true},
		// sudomains should work too
		{"about.google.com", true},
		{"gobblety gook", false},
		{"modeep.org/notreal", false},
	}

	for _, u := range test_urls {
		_, err := GetResolvedURLResponse(u.URL)
		if u.Valid && err != nil {
			t.Fatal(err)
		} else if !u.Valid && err == nil {
			t.Fatalf("expected error for url %s", u.URL)
		}
	}
}

func TestHeadersAreApplied(t *testing.T) {
	var expected_headers = map[string]string{
		"User-Agent": MODEEP_BOT_USER_AGENT,
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate, br",
	}
	test_server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				for k, v := range expected_headers {
					if r.Header.Get(k) != v {
						t.Errorf(
							"Expected header %s to be %s, got %s",
							k,
							v,
							r.Header.Get(k),
						)
					}
				}
			},
		))
	defer test_server.Close()

	resp, err := GetResolvedURLResponse(test_server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if _, err = io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

}

// this has to be valid and receive a reply back from a test response or
// GetLinkExtraMetadataFromHTML() will not set it
const TEST_PREVIEW_IMAGE_URL = "https://github.com/julianlk522/modeep-frontend/raw/main/public/home.webp"
func TestGetLinkExtraMetadataFromHTML(t *testing.T) {
	mock_metas := []HTMLMetadata{
		// Auto Summary should be og:description,
		// Preview image should be set
		{
			Title:         "title",
			Desc:          "description",
			OGTitle:       "og:title",
			OGDesc:        "og:description",
			OGImage:       TEST_PREVIEW_IMAGE_URL,
			OGAuthor:      "",
			OGPublisher:   "",
			OGSiteName:    "og:site_name",
		},
		// Auto Summary should be description
		{
			Title:         "",
			Desc:          "description",
			OGTitle:       "",
			OGDesc:        "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
		},
		// Auto Summary should be og:title
		{
			Title:         "title",
			Desc:          "",
			OGTitle:       "og:title",
			OGDesc:        "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
		},
		// Auto Summary should be title
		{
			Title:         "title",
			Desc:          "",
			OGTitle:       "",
			OGDesc:        "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "test",
			OGPublisher:   "",
		},
		// Auto Summary should be test
		// Preview image should be set
		{
			Title:         "",
			Desc:          "",
			OGTitle:       "",
			OGDesc:        "",
			OGImage:       TEST_PREVIEW_IMAGE_URL,
			OGAuthor:      "",
			OGSiteName:    "test",
			OGPublisher:   "",
		},
		// Auto Summary should be twitter:desc
		// Preview image should be set
		{
			Title:         "",
			Desc:          "",
			OGTitle:       "",
			OGDesc:        "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
			TwitterTitle:  "twitter:title",
			TwitterDesc:   "twitter:desc",
			TwitterImage:  TEST_PREVIEW_IMAGE_URL,
		},
		// Auto Summary should be twitter:title
		// Preview image should be set
		{
			Title:         "",
			Desc:          "",
			OGTitle:       "",
			OGDesc:        "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
			TwitterTitle:  "twitter:title",
			TwitterDesc:   "",
			TwitterImage:  TEST_PREVIEW_IMAGE_URL,
		},
	}

	for i, meta := range mock_metas {
		x_md := getLinkExtraMetadataFromHTML(&url.URL{}, meta)

		switch i {
			case 0:
				if x_md.AutoSummary != "og:description" {
					t.Fatalf("og:description provided but auto summary set to: %s", x_md.AutoSummary)
				} 
				if x_md.PreviewImgURL != mock_metas[0].OGImage {
					t.Fatalf(
						"expected og:image to be set to %s, got %s",
						mock_metas[0].OGImage,
						x_md.PreviewImgURL,
					)
				}
			case 1:
				if x_md.AutoSummary != "description" {
					t.Fatalf("description provided but auto summary set to: %s", x_md.AutoSummary)
				}
			case 2:
				if x_md.AutoSummary != "og:title" {
					t.Fatalf("og:title provided but auto summary set to: %s", x_md.AutoSummary)
				}
			case 3:
				if x_md.AutoSummary != "title" {
					t.Fatalf("title provided but auto summary set to: %s", x_md.AutoSummary)
				}
			case 4:
				if x_md.AutoSummary != "test" {
					t.Fatalf("og:sitename provided but auto summary set to: %s", x_md.AutoSummary)
				} 

				if x_md.PreviewImgURL != mock_metas[4].OGImage {
					t.Fatalf(
						"expected og:image to be set to %s, got %s",
						mock_metas[4].OGImage,
						x_md.PreviewImgURL,
					)
				}
			case 5:
				if x_md.AutoSummary != "twitter:desc" {
					t.Fatalf("twitter:desc provided but auto summary set to: %s", x_md.AutoSummary)	
				} 

				if x_md.PreviewImgURL != mock_metas[5].TwitterImage {
					t.Fatalf(
						"expected twitter:image to be set to %s, got %s",
						mock_metas[5].TwitterImage,
						x_md.PreviewImgURL,
					)
				}
			case 6:
				if x_md.AutoSummary != "twitter:title" {
					t.Fatalf("twitter:title provided but auto summary set to: %s", x_md.AutoSummary)	
				}
			default:
				t.Fatal("unhandled case, you f'ed up")
		}
	}
}

func TestLinkAlreadyAdded(t *testing.T) {
	var test_urls = []struct {
		URL   string
		Added bool
	}{
		{"https://stackoverflow.co/", true},
		{"https://www.ronjarzombek.com", true},
		{"https://somethingnotonmodeep", false},
		{"jimminy jillickers", false},
	}

	for _, u := range test_urls {
		added, _ := LinkAlreadyAdded(u.URL)
		if u.Added && !added {
			t.Fatalf("expected url %s to be added", u.URL)
		} else if !u.Added && added {
			t.Fatalf("%s NOT added, expected error", u.URL)
		}
	}
}

func TestIncrementSpellfixRanksForCats(t *testing.T) {
	var test_cats = []struct {
		Cats         []string
		CurrentRanks []int
	}{
		{
			[]string{"umvc3"},
			[]int{4},
		},
		{
			[]string{"flowers", "nerd"},
			[]int{6, 1},
		},
		// cat doesn't exist: should be added to global_cats_spellfix
		{
			[]string{"jksfdkhsdf"},
			[]int{0},
		},
	}

	for _, tc := range test_cats {
		err := IncrementSpellfixRanksForCats(nil, tc.Cats)
		if err != nil {
			t.Fatal(err)
		}

		for i, cat := range tc.Cats {
			var rank int
			err := db.Client.QueryRow(
				"SELECT rank FROM global_cats_spellfix WHERE word = ?", cat,
			).Scan(&rank)

			if err != nil {
				t.Fatal(err)
			} else if rank != tc.CurrentRanks[i]+1 {
				t.Fatal(
					"expected rank for", cat, "to be", tc.CurrentRanks[i]+1, "got", rank,
				)
			}
		}
	}
}

// Delete link
func TestDecrementSpellfixRanksForCats(t *testing.T) {
	var test_cats = []struct {
		Cats         []string
		CurrentRanks []int
	}{
		{
			[]string{"test"},
			[]int{21},
		},
		{
			[]string{"coding", "hacking"},
			[]int{6, 3},
		},
	}

	for _, tc := range test_cats {
		err := DecrementSpellfixRanksForCats(nil, tc.Cats)
		if err != nil {
			t.Fatal(err)
		}

		for i, cat := range tc.Cats {
			var rank int
			err := db.Client.QueryRow(
				"SELECT rank FROM global_cats_spellfix WHERE word = ?", cat,
			).Scan(&rank)

			if err != nil {
				t.Fatal(err)
			} else if rank != tc.CurrentRanks[i]-1 {
				t.Fatal(
					"expected rank for", cat, "to be", tc.CurrentRanks[i]-1, "got", rank,
				)
			}
		}
	}
}

// Star/unstar link
func TestUserSubmittedLink(t *testing.T) {
	var test_links = []struct {
		ID                  string
		SubmittedByTestUser bool
	}{
		// user jlk submitted links with ID 13, 23
		// (not 0 or 1)
		{"7", false},
		{"13", true},
		{"23", true},
		{"0", false},
		{"1", false},
	}

	for _, l := range test_links {
		if got := UserSubmittedLink(TEST_LOGIN_NAME, l.ID); got != l.SubmittedByTestUser {
			t.Fatalf("expected %t, got %t for link %s", l.SubmittedByTestUser, got, l.ID)
		}
	}
}

func TestUserHasStarredLink(t *testing.T) {
	var test_links = []struct {
		ID              string
		StarredByTestUser bool
	}{
		// user jlk starred links with ID 24, 32, 103
		// (not 9, 11, or 15)
		{"24", true},
		{"32", true},
		{"103", true},
		{"9", false},
		{"11", false},
		{"15", false},
	}

	for _, l := range test_links {
		if got := UserHasStarredLink(TEST_USER_ID, l.ID); got != l.StarredByTestUser {
			t.Fatalf("expected %t, got %t", l.StarredByTestUser, got)
		}
	}
}
