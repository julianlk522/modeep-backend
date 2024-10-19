package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mattn/go-sqlite3"
)

var (
	Client *sql.DB
)

const AUTO_SUMMARY_USER_ID = "ca39e263-2ac7-4d70-abc5-b9b8f1bff332"

var _, db_file, _, _ = runtime.Caller(0)
var db_dir = filepath.Dir(db_file)

func init() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
}

func Connect() error {
	LoadSpellfix()

	var err error
	Client, err = sql.Open("sqlite-spellfix1", db_dir+"/fitm.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=100000000")
	if err != nil {
		return err
	}

	err = Client.Ping()
	if err != nil {
		return err
	}

	log.Print("DB connection verified")

	return nil
}

func LoadSpellfix() {
	var spellfix_path string

	// check for FITM_TEST_DATA_PATH env var
	// if not set, use default path
	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		log.Print("FITM_TEST_DATA_PATH not set, attempting find spellfix at default path")
		spellfix_path = filepath.Join(db_dir, "spellfix")

	} else {
		log.Print("attempting to find spellfix at FITM_TEST_DATA_PATH")
		spellfix_path = test_data_path + "/spellfix"
	}
	sql.Register(
		"sqlite-spellfix1",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(c *sqlite3.SQLiteConn) error {
				return c.LoadExtension(spellfix_path, "sqlite3_spellfix_init")
			},
		},
	)
	log.Print("Loaded spellfix")
}
