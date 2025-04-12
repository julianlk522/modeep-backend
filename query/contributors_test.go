package query

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/julianlk522/fitm/model"
)

func TestNewTopContributors(t *testing.T) {
	contributors_sql := NewTopContributors()
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	rows, err := TestClient.Query(contributors_sql.Text, contributors_sql.Args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}

	if len(cols) == 0 {
		t.Fatal("no columns")
	} else if len(cols) != 2 {
		t.Fatalf("wrong columns (got %d, want 2)", len(cols))
	}

	var test_cols = []struct {
		Want string
	}{
		{"count"},
		{"submitted_by"},
	}

	for i, col := range cols {
		if col.Name() != test_cols[i].Want {
			t.Fatalf("column %d: got %s, want %s", i, col.Name(), test_cols[i].Want)
		}
	}
}

func TestTopContributorsFromCats(t *testing.T) {
	contributors_sql := NewTopContributors().FromCats(
		[]string{
			"umvc3",
			"c. viper",
		},
	)

	contributors_sql.Text = strings.Replace(
		contributors_sql.Text,
		`SELECT
count(l.id) as count, l.submitted_by
FROM Links l`,
		`SELECT
count(l.id) as count, l.global_cats
FROM Links l`,
		1)

	rows, err := TestClient.Query(contributors_sql.Text, contributors_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var cat, count string
		if err := rows.Scan(&count, &cat); err != nil {
			t.Fatal(err)
		} else if !strings.Contains(strings.ToLower(cat), "umvc3") {
			t.Fatalf("got %s, should contain %s", cat, "umvc3")
		}
	}
}

func TestTopContributorsWithURLContaining(t *testing.T) {
	contributors_sql := NewTopContributors().WithURLContaining("google")
	if contributors_sql.Error != nil {
		t.Fatal(contributors_sql.Error)
	}

	rows, err := TestClient.Query(contributors_sql.Text, contributors_sql.Args...)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	defer rows.Close()

	var contributors []model.Contributor
	for rows.Next() {
		var c model.Contributor
		if err := rows.Scan(&c.LinksSubmitted, &c.LoginName); err != nil {
			t.Fatal(err)
		}
		contributors = append(contributors, c)
	}

	if len(contributors) == 0 {
		t.Fatal("no contributors")
	}

	// verify counts
	for _, c := range contributors {
		var count int
		if err := TestClient.QueryRow(`SELECT count(id) as count 
		FROM LINKS 
		WHERE url LIKE ?
		AND submitted_by = ?`,
			"%google%",
			c.LoginName,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if c.LinksSubmitted != count {
			t.Fatalf("expected %d, got %d", c.LinksSubmitted, count)
		}
	}
}

func TestTopContributorsDuringPeriod(t *testing.T) {
	var test_periods = [7]struct {
		Period string
		Valid  bool
	}{
		{"day", true},
		{"week", true},
		{"month", true},
		{"year", true},
		{"all", false},
		{"shouldfail", false},
	}

	// Period only
	for _, period := range test_periods {
		contributors_sql := NewTopContributors().DuringPeriod(period.Period)
		if period.Valid && contributors_sql.Error != nil {
			t.Fatal(contributors_sql.Error)
		} else if !period.Valid && contributors_sql.Error == nil {
			t.Fatalf("expected error for period %s", period.Period)
		}

		rows, err := TestClient.Query(contributors_sql.Text, contributors_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		defer rows.Close()
	}

	// Period and Cats
	for _, period := range test_periods {
		contributors_sql := NewTopContributors().DuringPeriod(period.Period).FromCats([]string{"umvc3"})
		if period.Valid && contributors_sql.Error != nil {
			t.Fatal(contributors_sql.Error)
		} else if !period.Valid && contributors_sql.Error == nil {
			t.Fatalf("expected error for period %s", period.Period)
		}

		rows, err := TestClient.Query(contributors_sql.Text, contributors_sql.Args...)
		if err != nil && err != sql.ErrNoRows {
			t.Fatal(err)
		}
		defer rows.Close()
	}
}
