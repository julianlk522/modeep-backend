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

func TestGetCatsWithEscapedReservedChars(t *testing.T) {
	var test_cats = struct {
		Cats            []string
		ExpectedResults []string
	}{
		Cats: []string{
			"c. vi.per",
			"hsien-ko",
			"Ian's House",
			"#hashtag",
			"dolla$",
			"per%cent",
			"A&W",
			"back\\slash",
			"slash/slash/slash",
			"func(",
			"func)",
			"bra[",
			"ckets]",
			"bra{",
			"ces}",
			"either|or",
			"colon:colon",
			"Steins;Gate",
			"=3",
			"question?question",
			"goober@mail",
		},
		ExpectedResults: []string{
			`c"." vi"."per`,
			`hsien"-"ko`,
			`Ian"'"s House`,
			`"#"hashtag`,
			`dolla"$"`,
			`per"%"cent`,
			`A"&"W`,
			`back"\"slash`,
			`slash"/"slash"/"slash`,
			`func"("`,
			`func")"`,
			`bra"["`,
			`ckets"]"`,
			`bra"{"`,
			`ces"}"`,
			`either"|"or`,
			`colon":"colon`,
			`Steins";"Gate`,
			`"="3`,
			`question"?"question`,
			`goober"@"mail`,
		},
	}
	escaped_cats := GetCatsSurroundedInDoubleQuotes(test_cats.Cats)
	for i, res := range escaped_cats {
		if res != test_cats.ExpectedResults[i] {
			t.Fatalf("got %s, want %s", test_cats.Cats[i], test_cats.ExpectedResults[i])
		}
	}
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
