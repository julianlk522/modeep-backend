package dbtest

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/julianlk522/fitm/db"
	_ "github.com/mattn/go-sqlite3"
)

func SetupTestDB() error {
	log.Print("setting up test DB client")

	// create in-memory DB connection
	TestClient, err := sql.Open("sqlite-spellfix1", "file::memory:?cache=shared")
	if err != nil {
		return fmt.Errorf("could not open in-memory DB: %s", err)
	}

	var sql_dump_path string

	// check for FITM_TEST_DATA_PATH env var,
	// if not set, use default path
	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		log.Printf("FITM_TEST_DATA_PATH not set, using default path")
		_, dbtest_file, _, _ := runtime.Caller(0)
		dbtest_dir := filepath.Dir(dbtest_file)
		db_dir := filepath.Join(dbtest_dir, "../db")
		sql_dump_path = filepath.Join(db_dir, "fitm_test.db.sql")
	} else {
		log.Print("using FITM_TEST_DATA_PATH")
		sql_dump_path = test_data_path + "/fitm_test.db.sql"
	}

	sql_dump, err := os.ReadFile(sql_dump_path)
	if err != nil {
		return err
	}
	_, err = TestClient.Exec(string(sql_dump))
	if err != nil {
		return err
	}

	// verify that in-memory DB has new test data
	var link_id string
	err = TestClient.QueryRow("SELECT id FROM Links WHERE id = '1';").Scan(&link_id)
	if err != nil {
		return fmt.Errorf("in-memory DB did not receive dump data: %s", err)
	}
	log.Printf("verified dump data added to test DB")

	// verify that in-memory DB has spellfix1
	if _, err = TestClient.Exec(`SELECT word, rank FROM global_cats_spellfix;`); err != nil {
		return err
	}

	// switch DB client to TestClient
	db.Client = TestClient
	log.Print("switched to test DB client")

	return nil
}
