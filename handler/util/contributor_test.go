package handler

import (
	"net/url"
	"testing"

	"github.com/julianlk522/modeep/model"
	"github.com/julianlk522/modeep/query"
)

func TestGetContributorsOptionsFromRequestParams(t *testing.T) {
	var test_params = []model.TopContributorsOptions{
		{
			CatFiltersWithSpellingVariants: []string{"umvc3"},
		},
		{
			NeuteredCatFilters: []string{"test"},
		},
		{
			SummaryContains: "test",
		},
		{
			URLContains: "test",
		},
		{
			URLLacks: "test",
		},
		{
			Period: "day",
		},
	}

	for _, tp := range test_params {
		if _, err := GetTopContributorsOptionsFromRequestParams(url.Values{
			"cats":         tp.CatFiltersWithSpellingVariants,
			"neutered":     tp.NeuteredCatFilters,
			"url_contains": []string{tp.URLContains},
			"url_lacks":    []string{tp.URLLacks},
			"period":       []string{string(tp.Period)},
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewTopContributors(t *testing.T) {
	contributors_sql := query.NewTopContributors()
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	contributors := ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no rows")
	}
}

func TestTopContributorsFromOptions(t *testing.T) {
	// single cat
	test_cats_str := test_single_cat[0]
	opts := &model.TopContributorsOptions{
		CatFiltersWithSpellingVariants: []string{test_cats_str},
	}
	contributors_sql, err := query.NewTopContributors().FromOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	contributors := ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no rows")
	}

	// Verify number of submitted links for each contributor
	var ls int
	for _, contributor := range *contributors {
		err := TestClient.QueryRow(`SELECT count(*)
				FROM Links
				WHERE submitted_by = ?
				AND ',' || global_cats || ',' LIKE '%' || ? || '%'`,
			contributor.LoginName,
			test_cats_str).Scan(&ls)
		if err != nil {
			t.Fatal(err)
		} else if ls != contributor.LinksSubmitted {
			t.Fatalf(
				"expected %d links submitted, got %d (contributor: %s)",
				contributor.LinksSubmitted,
				ls,
				contributor.LoginName,
			)
		}
	}

	// multiple cats
	opts = &model.TopContributorsOptions{
		CatFiltersWithSpellingVariants: test_multiple_cats,
	}
	contributors_sql, err = query.NewTopContributors().FromOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	contributors = ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no rows")
	}

	// Verify number of submitted links for each contributor
	for _, contributor := range *contributors {
		err := TestClient.QueryRow(`SELECT count(*)
				FROM Links
				WHERE submitted_by = ?
				AND ',' || global_cats || ',' LIKE '%' || ? || '%'
				AND ',' || global_cats || ',' LIKE '%' || ? || '%';`,
			contributor.LoginName,
			test_multiple_cats[0],
			test_multiple_cats[1]).Scan(&ls)
		if err != nil {
			t.Fatal(err)
		} else if ls != contributor.LinksSubmitted {
			t.Fatalf(
				"expected %d links submitted, got %d (contributor: %s)",
				contributor.LinksSubmitted,
				ls,
				contributor.LoginName,
			)
		}
	}
}
