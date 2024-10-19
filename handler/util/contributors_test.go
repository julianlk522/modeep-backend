package handler

import (
	"testing"

	"github.com/julianlk522/fitm/query"
)

func TestScanContributors(t *testing.T) {

	// no cats
	contributors_sql := query.NewContributors()
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	contributors := ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no contributors")
	}

	// single cat
	contributors_sql = query.NewContributors().FromCats(test_single_cat)
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	test_cats_str := test_single_cat[0]
	contributors = ScanContributors(contributors_sql)
	if len(*contributors) == 0 {
		t.Fatal("no contributors")
	}

	// verify that each contributor submitted the correct number of links
	var ls int
	for _, contributor := range *contributors {
		err := TestClient.QueryRow(`SELECT count(*)
				FROM Links
				WHERE submitted_by = ?
				AND ',' || global_cats || ',' LIKE '%,' || ? || ',%'`,
				contributor.LoginName,
				test_cats_str).Scan(&ls)
		if err != nil {
			t.Fatal(err)
		} else if ls != contributor.LinksSubmitted {
			t.Fatalf(
				"expected %d links submitted, got %d (contributor: %s)", contributor.LinksSubmitted,
				ls,
				contributor.LoginName,
			)
		}
	}

	// multiple cats
	contributors_sql = query.NewContributors().FromCats(test_multiple_cats)
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}
	
	contributors = ScanContributors(contributors_sql)

	if len(*contributors) == 0 {
		t.Fatal("no contributors")
	}

	// verify that each contributor submitted the correct number of links
	for _, contributor := range *contributors {
		err := TestClient.QueryRow(`SELECT count(*)
				FROM Links
				WHERE submitted_by = ?
				AND ',' || global_cats || ',' LIKE '%,' || ? || ',%'
				AND ',' || global_cats || ',' LIKE '%,' || ? || ',%';`,
				contributor.LoginName,
				test_multiple_cats[0],
				test_multiple_cats[1]).Scan(&ls)
		if err != nil {
			t.Fatal(err)
		} else if ls != contributor.LinksSubmitted {
			t.Fatalf(
				"expected %d links submitted, got %d (contributor: %s)", contributor.LinksSubmitted,
				ls,
				contributor.LoginName,
			)
		}
	}
}
