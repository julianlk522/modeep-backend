package query

import (
	"database/sql"
	"testing"

	"github.com/julianlk522/modeep/model"
)

func TestNewTopContributors(t *testing.T) {
	contributors_sql := NewTopContributors()
	rows, err := contributors_sql.ValidateAndExecuteRows()
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

func TestTopContributorsFromCatFilters(t *testing.T) {
	test_cat_filters := []string{"test"}
	contributors_sql := NewTopContributors().fromCatFilters(test_cat_filters)
	rows, err := contributors_sql.ValidateAndExecuteRows()
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf(
			"got %v, SQL was %s",
			err,
			contributors_sql.Text,
		)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no rows")
	}

	for rows.Next() {
		var cat, count string
		if err := rows.Scan(&count, &cat); err != nil {
			t.Fatal(err)
		}
	}

	// TODO confirm counts
}

func TestTopContributorsFromNeuteredCatFilters(t *testing.T) {
	var test_neutered_cats = []string{"test", "best"}
	contributors_sql := NewTopContributors().fromNeuteredCatFilters(test_neutered_cats)
	rows, err := contributors_sql.ValidateAndExecuteRows()
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf(
			"got %v, SQL was %s",
			err,
			contributors_sql.Text,
		)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("no rows")
	}

	for rows.Next() {
		var cat, count string
		if err := rows.Scan(&count, &cat); err != nil {
			t.Fatal(err)
		}
	}

	// TODO confirm counts
}

func TestTopContributorsWhereGlobalSummaryContains(t *testing.T) {
	// case-insensitive
	contributors_sql := NewTopContributors().whereGlobalSummaryContains("gooGLE")
	rows, err := contributors_sql.ValidateAndExecuteRows()
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
		WHERE global_summary LIKE ?
		AND submitted_by = ?`,
			"%google%",
			c.LoginName,
		).Scan(&count); err != nil {
			t.Fatal(err)
		} else if count != c.LinksSubmitted {
			t.Fatalf("got %d, want %d", count, c.LinksSubmitted)
		}
	}

	// no conflict w/ other methods
	contributors_sql = NewTopContributors().
		whereGlobalSummaryContains("TEST").
		whereURLContains("www").
		whereURLLacks("test").
		duringPeriod("all")
	rows, err = contributors_sql.ValidateAndExecuteRows()
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf(
			"got %v, SQL: %s, args: %v",
			err,
			contributors_sql.Text,
			contributors_sql.Args,
		)
	}
	defer rows.Close()

	contributors = []model.Contributor{}
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

	for _, c := range contributors {
		var count int
		if err := TestClient.QueryRow(`SELECT count(id) as count 
		FROM LINKS 
		WHERE global_summary LIKE ?
		AND url LIKE ?
		AND url NOT LIKE ?
		AND submitted_by = ?`,
			"%test%",
			"%www%",
			"%test%",
			c.LoginName,
		).Scan(&count); err != nil {
			t.Fatal(err)
		} else if count != c.LinksSubmitted {
			t.Fatalf("got %d, want %d", count, c.LinksSubmitted)
		}
	}
}

func TestTopContributorsWhereURLContains(t *testing.T) {
	// case-insensitive
	contributors_sql := NewTopContributors().whereURLContains("gooGLE")
	rows, err := contributors_sql.ValidateAndExecuteRows()
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
	var test_periods = []struct {
		Period model.Period
		Valid  bool
	}{
		{"day", true},
		{"week", true},
		{"month", true},
		{"year", true},
		{"all", true},
		{"shouldfail", false},
	}

	// Period only
	for _, period := range test_periods {
		contributors_sql := NewTopContributors().duringPeriod(period.Period)
		if period.Valid && contributors_sql.Error != nil {
			t.Fatal(contributors_sql.Error)
		} else if !period.Valid && contributors_sql.Error == nil {
			t.Fatalf("expected error for period %s", period.Period)
		}

		if period.Valid {
			if _, err := contributors_sql.ValidateAndExecuteRows(); err != nil && err != sql.ErrNoRows {
				t.Fatalf("got %v, SQL: %s, args: %v", err, contributors_sql.Text, contributors_sql.Args)
			}
		}
	}

	// Period and cat filter
	for _, period := range test_periods {
		contributors_sql := NewTopContributors().duringPeriod(period.Period).fromCatFilters([]string{"umvc3"})
		if period.Valid && contributors_sql.Error != nil {
			t.Fatal(contributors_sql.Error)
		} else if !period.Valid && contributors_sql.Error == nil {
			t.Fatalf("expected error for period %s", period.Period)
		}

		if period.Valid {
			if _, err := contributors_sql.ValidateAndExecuteRows(); err != nil && err != sql.ErrNoRows {
				t.Fatal(err)
			}
		}
	}
}
