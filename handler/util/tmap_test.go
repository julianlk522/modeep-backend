package handler

import (
	"slices"
	"strings"
	"testing"

	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func TestUserExists(t *testing.T) {
	var test_login_names = []struct {
		login_name string
		Exists     bool
	}{
		{"johndoe", false},
		{"janedoe", false},
		{TEST_LOGIN_NAME, true},
	}

	for _, l := range test_login_names {
		got, err := UserExists(l.login_name)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.Exists != got {
			t.Fatalf("expected %t, got %t", l.Exists, got)
		}
	}
}

func TestBuildTmapFromOpts(t *testing.T) {
	var test_opts = []struct {
		LoginName        string
		RequestingUserID string
		CatsParams       string
		SortBy           string
		IncludeNSFW      bool
		SectionParams    string
		PageParams       int
		Valid            bool
	}{
		{TEST_LOGIN_NAME, TEST_USER_ID, "", "times_starred", false, "", 1, true},
		{TEST_LOGIN_NAME, TEST_REQ_USER_ID, "", "times_starred", true, "", 1, true},
		{TEST_LOGIN_NAME, "", "", "newest", true, "", 1, true},
		{TEST_LOGIN_NAME, TEST_USER_ID, "umvc3", "newest", true, "", 1, true},
		{TEST_LOGIN_NAME, TEST_REQ_USER_ID, "", "oldest", false, "", 0, true},
		{TEST_LOGIN_NAME, "", "", "times_starred", false, "", 10, true},
		{TEST_LOGIN_NAME, TEST_USER_ID, "umvc3,flowers", "oldest", true, "", 1, true},
		{TEST_LOGIN_NAME, "", "umvc3,flowers", "times_starred", false, "", 2, true},
		{TEST_LOGIN_NAME, "", "umvc3,flowers", "", true, "", 1, true},
		{TEST_LOGIN_NAME, "", "umvc3,flowers", "", true, "submitted", 4, true},
		{TEST_LOGIN_NAME, "", "umvc3,flowers", "oldest", true, "starred", 0, true},
		{TEST_LOGIN_NAME, "", "umvc3,flowers", "clicks", true, "starred", 1, true},
		// "notasection" is invalid
		{TEST_LOGIN_NAME, "", "umvc3,flowers", "oldest", true, "notasection", 1, false},
		// negative page is invalid
		{TEST_LOGIN_NAME, "", "", "newest", true, "submitted", -1, false},
	}

	for _, td := range test_opts {
		var opts = &model.TmapOptions{
			OwnerLoginName: td.LoginName,
			RawCatsParams:  td.CatsParams,
			AsSignedInUser: td.RequestingUserID,
			SortBy:         td.SortBy,
			IncludeNSFW:    td.IncludeNSFW,
			Section:        td.SectionParams,
			Page:           td.PageParams,
		}

		if td.CatsParams != "" {
			cats := query.GetCatsOptionalPluralOrSingularForms(
				strings.Split(td.CatsParams, ","),
			)
			opts.Cats = cats
		}

		var tmap any
		var err error

		if td.RequestingUserID != "" {
			tmap, err = BuildTmapFromOpts[model.TmapLinkSignedIn](opts)
		} else {
			tmap, err = BuildTmapFromOpts[model.TmapLink](opts)
		}

		if (err == nil) != td.Valid {
			t.Fatalf("expected %t, got error %s", td.Valid, err)
		}

		if !td.Valid {
			continue
		}

		// verify type and filtered
		var is_filtered bool
		switch tmap.(type) {
		case model.TmapWithProfilePage[model.TmapLink], model.TmapWithProfilePage[model.TmapLinkSignedIn]:
			is_filtered = false
		case 
			model.TmapWithCatFiltersPage[model.TmapLink],
			model.TmapWithCatFiltersPage[model.TmapLinkSignedIn],
			model.TmapIndividualSectionWithCatFiltersPage[model.TmapLink],
			model.TmapIndividualSectionWithCatFiltersPage[model.TmapLinkSignedIn]:
			is_filtered = true
		case 
			model.TmapPage[model.TmapLink],
			model.TmapPage[model.TmapLinkSignedIn],
			model.TmapIndividualSectionPage[model.TmapLink],
			model.TmapIndividualSectionPage[model.TmapLinkSignedIn]:
			continue
		}

		if is_filtered && td.CatsParams == "" {
			t.Fatalf("expected unfiltered treasure map type, got %T", tmap)
		} else if !is_filtered && td.CatsParams != "" {
			t.Fatalf("expected filtered treasure map type, got %T (request params: %+v)", tmap, td)
		}
	}
}

func TestBuildTmapLinksQueryAndScan(t *testing.T) {
	var test_options = []model.TmapOptions{
		{
			OwnerLoginName: TEST_LOGIN_NAME,
			AsSignedInUser: TEST_USER_ID,
		},
		{
			OwnerLoginName: TEST_LOGIN_NAME,
			AsSignedInUser: TEST_REQ_USER_ID,
			Cats:           []string{"umvc3"},
			Period:         "year",
			SortBy:         "newest",
			IncludeNSFW:    true,
		},
		{
			OwnerLoginName: TEST_LOGIN_NAME,
			SummaryContains: "web",
			URLContains:     "com",
			URLLacks:        "net",
		},
	}

	for _, to := range test_options {
		if to.AsSignedInUser != "" {
			if _, err := buildTmapLinksQueryAndScan[model.TmapLinkSignedIn](
				query.NewTmapSubmitted(to.OwnerLoginName),
				&to,
			); err != nil {
				t.Fatal(err)
			}

			if _, err := buildTmapLinksQueryAndScan[model.TmapLinkSignedIn](
				query.NewTmapStarred(to.OwnerLoginName),
				&to,
			); err != nil {
				t.Fatal(err)
			}

			if _, err := buildTmapLinksQueryAndScan[model.TmapLinkSignedIn](
				query.NewTmapTagged(to.OwnerLoginName),
				&to,
			); err != nil {
				t.Fatal(err)
			}
		} else {
			if _, err := buildTmapLinksQueryAndScan[model.TmapLink](
				query.NewTmapSubmitted(to.OwnerLoginName),
				&to,
			); err != nil {
				t.Fatal(err)
			}

			if _, err := buildTmapLinksQueryAndScan[model.TmapLink](
				query.NewTmapStarred(to.OwnerLoginName),
				&to,
			); err != nil {
				t.Fatal(err)
			}

			if _, err := buildTmapLinksQueryAndScan[model.TmapLink](
				query.NewTmapTagged(to.OwnerLoginName),
				&to,
			); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestScanTmapLinks(t *testing.T) {
	var test_options = []model.TmapOptions{
		{
			OwnerLoginName: TEST_LOGIN_NAME,
			AsSignedInUser: TEST_USER_ID,
		},
		{
			OwnerLoginName: TEST_LOGIN_NAME,
			AsSignedInUser: TEST_REQ_USER_ID,
		},
		{
			OwnerLoginName: TEST_LOGIN_NAME,
		},
	}

	for _, to := range test_options {
		submitted_sql := query.NewTmapSubmitted(to.OwnerLoginName).FromOptions(&to).Build()
		starred_sql := query.NewTmapStarred(to.OwnerLoginName).FromOptions(&to).Build()
		tagged_sql := query.NewTmapTagged(to.OwnerLoginName).FromOptions(&to).Build()

		for _, sql := range []*query.Query{submitted_sql, starred_sql, tagged_sql} {
			var err error
			if to.AsSignedInUser != "" {
				_, err = scanTmapLinks[model.TmapLinkSignedIn](sql)
			} else {
				_, err = scanTmapLinks[model.TmapLink](sql)
			}
			if err != nil {
				t.Fatalf("failed with error %s", err)
			}
		}

	}
}

func TestGetCatCountsFromTmapLinks(t *testing.T) {
	tmap, err := BuildTmapFromOpts[model.TmapLink](&model.TmapOptions{
		OwnerLoginName: "xyz",
	})
	if err != nil {
		t.Fatalf("failed with error %s", err)
	}

	var all_links any

	switch tmap.(type) {
	case model.TmapWithProfilePage[model.TmapLink]:
		all_links = slices.Concat(
			*tmap.(model.TmapWithProfilePage[model.TmapLink]).Submitted,
			*tmap.(model.TmapWithProfilePage[model.TmapLink]).Starred,
			*tmap.(model.TmapWithProfilePage[model.TmapLink]).Tagged,
		)
		l, ok := all_links.([]model.TmapLink)
		if !ok {
			t.Fatalf("unexpected type %T", all_links)
		}

		// no omitted cats
		var unfiltered_test_cat_counts = []struct {
			Cat   string
			Count int32
		}{
			{"test", 2},
			// tag has cats "flowers" and "Flowers": tests that tags with
			// capitalization variant duplicates are only counted once still
			{"flowers", 2},
		}

		cat_counts := getCatCountsFromTmapLinks(&l, nil)
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

		// empty omitted cats
		// (should never happen, but should behave as if no omitted cats were passed)
		cat_counts = getCatCountsFromTmapLinks(
			&l,
			&model.TmapCatCountsOptions{
				RawCatsParams: "",
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

		// omitted cats
		var filtered_test_cat_counts = []struct {
			Cat   string
			Count int32
		}{
			{"test", 0},
			{"flowers", 2},
		}

		cat_counts = getCatCountsFromTmapLinks(
			&l,
			&model.TmapCatCountsOptions{
				RawCatsParams: "test",
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

func TestMergeCountsOfCatSpellingVariants(t *testing.T) {
	var counts = []model.CatCount{
		{Category: "Music", Count: 1},
		{Category: "music", Count: 1},  // should get added
		{Category: "musica", Count: 1}, // should NOT get added
		{Category: "Musics", Count: 5}, // should get added
		{Category: "musics", Count: 1}, // should get added
		{Category: "MODEEP", Count: 6},
		{Category: "modeep", Count: 5}, // should get added to above
	}
	mergeCountsOfCatSpellingVariants(&counts)

	// make sure the highest counts go first
	if counts[0].Category != "MODEEP" &&
		counts[1].Category != "Musics" &&
		counts[0].Count < counts[1].Count {
		t.Fatalf(
			"largest counts did not go first (counts were %+v)",
			counts,
		)
	}

	// make sure all the "music" variants added their counts
	if counts[1].Count != 1+1+5+1 {
		t.Fatalf(
			"expected count %d, got %d (counts were %+v)",
			8,
			counts[1].Count,
			counts,
		)
	}

	// make sure all like categories were merged
	// should be "Music", "musica", and "MODEEP" remaining
	if len(counts) != 3 {
		t.Fatalf(
			"expected count %d, got %d (counts were %+v)",
			3,
			len(counts),
			counts,
		)
	}
}

func TestCountTmapMergedCatsSpellingVariantsInLinksWithCatFilters(t *testing.T) {
	var test_links = []model.TmapLink{
		{Link: model.Link{
			Cats: "tests,Tests",
		}},
		{Link: model.Link{
			Cats: "Test",
		}},
		{Link: model.Link{
			Cats: "MoDeEp",
		}},
		{Link: model.Link{
			Cats: "testicles,modoop", // neither of these should be merged (that could get painful...)
		}},
	}
	var test_cat_filter = []string{
		"test",
"modeep",
	}
	var expected_merged_cats = []string{
		"tests", // pluralization variant
		"Test",  // capitalization variant
		"MoDeEp",
		"Tests", // pluralization and capitalization variant
	}

	got := countTmapMergedCatsSpellingVariantsInLinksFromCatFilters(
		&test_links,
		test_cat_filter,
	)

	if len(got) != len(expected_merged_cats) {
		t.Fatalf(
			"expected %d cats, got %d",
			len(expected_merged_cats),
			len(got),
		)
	}

	for _, cat := range got {
		if !slices.Contains(expected_merged_cats, cat) {
			t.Fatalf(
				"got cats %s, expected %s",
				got,
				expected_merged_cats,
			)
		}
	}
}

func TestScanTmapProfile(t *testing.T) {
	profile_sql := query.NewTmapProfile(TEST_LOGIN_NAME)
	profile, err := scanTmapProfile(profile_sql)
	if err != nil {
		t.Fatal(err)
	}

	if profile.LoginName != TEST_LOGIN_NAME {
		t.Fatalf(
			"expected %s, got %s", TEST_LOGIN_NAME,
			profile.LoginName,
		)
	}

	if profile.CreatedAt != "2024-04-10T03:48:09Z" {
		t.Fatalf(
			"expected %s, got %s", "2024-04-10T03:48:09Z",
			profile.CreatedAt,
		)
	}
}
