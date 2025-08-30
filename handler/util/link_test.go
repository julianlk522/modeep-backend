package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/julianlk522/modeep/db"
	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func TestPrepareLinksPage(t *testing.T) {
	var test_requests = []struct {
		LinksSQL   *query.TopLinks
		Options     *model.LinksPageOptions
		Valid      bool
	}{
		{
			LinksSQL:   query.NewTopLinks().Page(1),
			Options: &model.LinksPageOptions{},
			Valid:      true,
		},
		{
			LinksSQL:   query.NewTopLinks().FromCats([]string{"umvc3", "flowers"}).Page(1),
			Options: &model.LinksPageOptions{
				Cats: "umvc3,flowers",
			},
			Valid:      true,
		},
		{
			LinksSQL:   query.NewTopLinks().DuringPeriod("batman", "rating").Page(1),
			Options: &model.LinksPageOptions{
				NSFW: true,
			},
			Valid:      false,
		},
		{
			LinksSQL: &query.TopLinks{
				Query: query.Query{
					Text: "spiderman",
				},
			},
			Options: &model.LinksPageOptions{},
			Valid:      false,
		},
	}

	for _, tr := range test_requests {
		_, err := PrepareLinksPage[model.Link](tr.LinksSQL, tr.Options)
		if tr.Valid && err != nil {
			t.Fatal(err)
		} else if !tr.Valid && err == nil {
			t.Fatalf("expected error for request %+v\n", tr)
		}
	}
}

func TestScanLinks(t *testing.T) {
	links_sql := query.NewTopLinks()
	// NewTopLinks().Error tested in query/link_test.go

	// signed out
	links_page, err := ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links_page.Links) == 0 {
		t.Fatal("no links")
	}

	// signed in
	links_sql = links_sql.AsSignedInUser(TEST_REQ_USER_ID)
	signed_in_links_page, err := ScanRawLinksPageData[model.LinkSignedIn](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*signed_in_links_page.Links) == 0 {
		t.Fatal("no links")
	}
}

func TestPaginateLinks(t *testing.T) {

	// single page
	links_sql := query.NewTopLinks().FromCats([]string{"umvc3", "flowers"}).Page(1)
	links_page, err := ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	PaginateLinks(links_page.Links)
	if len(*links_page.Links) == 0 {
		t.Fatal("expected links")
	}

	// multiple pages
	links_sql = query.NewTopLinks().Page(1)
	links_page, err = ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links_page.Links) == 0 {
		t.Fatal("expected links")
	}

	PaginateLinks(links_page.Links)
	if len(*links_page.Links) == 0 {
		t.Fatal("expected links")
	}
}

func TestCountMergedCatSpellingVariants(t *testing.T) {
	// no links; no merged cats
	test_cat := "nonexistentcat"
	links_sql := query.NewTopLinks().FromCats([]string{test_cat}).DuringPeriod("day", "rating").Page(1)
	links_page, err := ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	PaginateLinks(links_page.Links)
	CountMergedCatSpellingVariants(links_page, test_cat)
	if len(links_page.MergedCats) != 0 {
		t.Fatal("expected no merged cats")
	}

	// 1 merged cat
	test_cat = "flower" // should merge "flowers"
	links_sql = query.NewTopLinks().FromCats([]string{test_cat})
	links_page, err = ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	PaginateLinks(links_page.Links)
	CountMergedCatSpellingVariants(links_page, test_cat)
	if len(links_page.MergedCats) != 1 {
		t.Fatalf("expected 1 merged cat, got %d (%v)", len(links_page.MergedCats), links_page.MergedCats)
	}

	// multiple merged cats
	test_cats := []string{"flower", "tests"} // should merge "flowers" and "test"
	links_sql = query.NewTopLinks().FromCats(test_cats)
	links_page, err = ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	PaginateLinks(links_page.Links)
	CountMergedCatSpellingVariants(links_page, strings.Join(test_cats, ","))
	if len(links_page.MergedCats) != 2 {
		t.Fatalf("expected 2 merged cats, got %d (%v)", len(links_page.MergedCats), links_page.MergedCats)
	}

	// inconsistent capitalization: should still merge
	test_cat = "FlOwEr" // should merge "flowers"
	links_sql = query.NewTopLinks().FromCats([]string{test_cat})
	links_page, err = ScanRawLinksPageData[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	PaginateLinks(links_page.Links)
	CountMergedCatSpellingVariants(links_page, test_cat)
	if len(links_page.MergedCats) != 1 {
		t.Fatalf("expected 1 merged cat, got %d (%v)", len(links_page.MergedCats), links_page.MergedCats)
	}

}

// Add link
func TestGetLinkExtraMetadataFromResponse(t *testing.T) {
	var test_links = []struct {
		new_link *model.NewLink
		Valid    bool
	}{
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "abc.com"}}, true},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "www.abc.com"}}, true},
		{&model.NewLink{NewLinkRequest: &model.NewLinkRequest{URL: "https://www.abc.com"}}, true},
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
		{"abc.com", true},
		{"www.abc.com", true},
		{"https://www.abc.com", true},
		{"about.google.com", true},
		{"julianlk.com/notreal", false},
		{"gobblety gook", false},
		// TODO: get the user agent headers to correctly apply and
		// add test case e.g., https://neal.fun/deep-sea
		// (responds with 403 if no user agent set)
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

func TestGetLinkExtraMetadataFromHTML(t *testing.T) {
	mock_metas := []HTMLMetadata{
		// Auto Summary should be og:description,
		// og:image should be set
		{
			Title:         "title",
			Description:   "description",
			OGTitle:       "og:title",
			OGDescription: "og:description",
			OGImage:       "https://i.ytimg.com/vi/L4gaqVH0QHU/maxresdefault.jpg",
			OGAuthor:      "",
			OGPublisher:   "",
			OGSiteName:    "og:site_name",
		},
		// Auto Summary should be description
		{
			Title:         "",
			Description:   "description",
			OGTitle:       "",
			OGDescription: "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
		},
		// Auto Summary should be og:title
		{
			Title:         "title",
			Description:   "",
			OGTitle:       "og:title",
			OGDescription: "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
		},
		// Auto Summary should be title
		{
			Title:         "title",
			Description:   "",
			OGTitle:       "",
			OGDescription: "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "test",
			OGPublisher:   "",
		},
		// Auto Summary should be test
		// og:image should be set
		{
			Title:         "",
			Description:   "",
			OGTitle:       "",
			OGDescription: "",
			OGImage:       "https://i.ytimg.com/vi/XdfoXdzGmr0/maxresdefault.jpg",
			OGAuthor:      "",
			OGSiteName:    "test",
			OGPublisher:   "",
		},
	}

	for i, meta := range mock_metas {
		x_md := GetLinkExtraMetadataFromHTML(meta)

		switch i {
		case 0:
			if x_md.AutoSummary != "og:description" {
				t.Fatalf("og:description provided but auto summary set to: %s", x_md.AutoSummary)
			} else if x_md.PreviewImgURL != "https://i.ytimg.com/vi/L4gaqVH0QHU/maxresdefault.jpg" {
				t.Fatal("expected og:image to be set")
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
			} else if x_md.PreviewImgURL != "https://i.ytimg.com/vi/XdfoXdzGmr0/maxresdefault.jpg" {
				t.Fatal("expected og:image to be set")
			}
		default:
			t.Fatal("unhandled case, you f'ed up dawg")
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

// Like / unlike link
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

func TestUserHasLikedLink(t *testing.T) {
	var test_links = []struct {
		ID              string
		LikedByTestUser bool
	}{
		// user jlk liked links with ID 24, 32, 103
		// (not 9, 11, or 15)
		{"24", true},
		{"32", true},
		{"103", true},
		{"9", false},
		{"11", false},
		{"15", false},
	}

	for _, l := range test_links {
		if got := UserHasLikedLink(TEST_USER_ID, l.ID); got != l.LikedByTestUser {
			t.Fatalf("expected %t, got %t", l.LikedByTestUser, got)
		}
	}
}

// Copy link
func TestUserHasCopiedLink(t *testing.T) {
	var test_links = []struct {
		ID               string
		CopiedByTestUser bool
	}{
		// test user jlk copied links with ID 19, 31, 32
		// (not 0, 1, or 104)
		{"19", true},
		{"31", true},
		{"32", true},
		{"0", false},
		{"1", false},
		{"104", false},
	}

	for _, l := range test_links {
		if got := UserHasCopiedLink(TEST_USER_ID, l.ID); got != l.CopiedByTestUser {
			t.Fatalf("expected %t, got %t", l.CopiedByTestUser, got)
		}
	}
}
