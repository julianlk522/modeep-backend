package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConnect(t *testing.T) {
	err := Client.Ping()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadSpellfix(t *testing.T) {
	// for some reason using the Client initialized in init() results in
	// "no such table: global_cats_spellfix"
	// so a temporary in-memory connection must be used instead

	// create in-memory DB connection
	TestClient, err := sql.Open("sqlite-spellfix1", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("could not open in-memory DB: %s", err)
	}

	var sql_dump_path string
	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		_, dbtest_file, _, _ := runtime.Caller(0)
		db_dir := filepath.Dir(dbtest_file)
		sql_dump_path = filepath.Join(db_dir, "fitm_test.db.sql")
	} else {
		sql_dump_path = test_data_path + "/fitm_test.db.sql"
	}

	sql_dump, err := os.ReadFile(sql_dump_path)
	if err != nil {
		t.Fatalf("could not read sql dump: %s", err)
	}
	_, err = TestClient.Exec(string(sql_dump))
	if err != nil {
		t.Fatalf("could not execute sql dump: %s", err)
	}

	var word, rank string
	if err := TestClient.QueryRow(`SELECT word, rank FROM global_cats_spellfix LIMIT 1;`).Scan(&word, &rank); err != nil {
		t.Fatal(err)
	}
}
