package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConnect(t *testing.T) {
	if err := Client.Ping(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSpellfix(t *testing.T) {
	// using the Client initialized in init() results in
	// "no such table: global_cats_spellfix" for some reason
	// so a temporary in-memory connection must be used instead

	TestClient, err := sql.Open("sqlite-spellfix1", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("could not open in-memory DB: %s", err)
	}

	var sql_dump_path, db_dir string

	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		_, dbtest_file, _, _ := runtime.Caller(0)
		db_dir = filepath.Dir(dbtest_file)
	} else {
		db_dir = test_data_path
	}
	sql_dump_path = filepath.Join(db_dir, "modeep_test.db.sql")

	sql_dump, err := os.ReadFile(sql_dump_path)
	if err != nil {
		t.Fatalf("could not read sql dump: %s", err)
	} else if _, err = TestClient.Exec(string(sql_dump)); err != nil {
		t.Fatalf("could not execute sql dump: %s", err)
	}

	var word, rank string
	if err := TestClient.QueryRow(`SELECT word, rank FROM global_cats_spellfix LIMIT 1;`).Scan(&word, &rank); err != nil {
		t.Fatal(err)
	}
}
