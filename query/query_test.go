package query

import (
	"database/sql"
	"log"
	"testing"

	"github.com/julianlk522/fitm/db"
	"github.com/julianlk522/fitm/dbtest"
)

var TestClient *sql.DB

func TestMain(m *testing.M) {
	if err := dbtest.SetupTestDB(); err != nil {
		log.Fatal(err)
	}
	// TestClient unneeded but helps to reiterate in tests that the DB connection is temporary in-memory
	TestClient = db.Client
	m.Run()
}

func TestWithOptionalPluralOrSingularForm(t *testing.T) {
	var test_cats = struct {
		Cats            []string
		ExpectedResults []string
	}{
		Cats: []string{
			"cat",
			"cats",
			"dress",
			"dresses",
			"iris",
			"irises",
			"music",
		},
		ExpectedResults: []string{
			`("cat" OR "cats")`,
			`("cats" OR "catses" OR "cat")`,
			`("dress" OR "dresses")`,
			`("dresses" OR "dress")`,
			`("iris" OR "irises" OR "iri")`,
			`("irises" OR "iriseses" OR "irise")`,
			`("music" OR "musics")`,
		},
	}

	for i, cat := range test_cats.Cats {
		cat = WithOptionalPluralOrSingularForm(cat)
		if cat != test_cats.ExpectedResults[i] {
			t.Fatalf("got %s, want %s", cat, test_cats.ExpectedResults[i])
		}
	}
}
