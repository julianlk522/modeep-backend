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
	Client, err = sql.Open("sqlite-spellfix1", db_dir+"/modeep.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=100000000")
	if err != nil {
		return err
	}

	if err = Client.Ping(); err != nil {
		return err
	}

	log.Print("DB connection verified")

	return nil
}

func LoadSpellfix() {
	var spellfix_path string

	backend_root_path := os.Getenv("MODEEP_BACKEND_ROOT")
	if backend_root_path == "" {
		log.Print("$MODEEP_BACKEND_ROOT not set, attempting to find spellfix at default path")
		spellfix_path = filepath.Join(db_dir, "spellfix")

	} else {
		test_data_path := backend_root_path + "/db"
		log.Printf("Attempting to find spellfix at %s", test_data_path + "/spellfix")
		spellfix_path = filepath.Join(test_data_path, "spellfix")
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
