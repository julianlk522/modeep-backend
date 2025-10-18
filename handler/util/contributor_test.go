package handler

import (
	"net/url"
	"strings"
	"testing"

	"github.com/julianlk522/modeep/query"
)

func TestScanContributors(t *testing.T) {
	// no cats
	contributors_sql := query.NewTopContributors()
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	contributors := ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no contributors")
	}

	// single cat
	contributors_sql = query.NewTopContributors().FromRequestParams(url.Values{"cats": []string{test_single_cat[0]}})
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	test_cats_str := test_single_cat[0]
	contributors = ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no contributors")
	}

	// verify each contributor submitted correct number of links
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
	multiple_cats_params := url.Values{"cats": []string{strings.Join(test_multiple_cats, ",")}}
	contributors_sql = query.NewTopContributors().FromRequestParams(multiple_cats_params)
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	contributors = ScanContributors(contributors_sql)

	if len(*contributors) == 0 {
		t.Fatal("no contributors")
	}

	// verify each contributor submitted correct number of links
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
