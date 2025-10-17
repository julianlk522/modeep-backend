package handler

import (
	"database/sql"
	"log"
	"testing"

	"github.com/julianlk522/modeep/db"
	"github.com/julianlk522/modeep/dbtest"
)

const (
	TEST_USER_ID    = "3"
	TEST_LOGIN_NAME = "jlk"
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