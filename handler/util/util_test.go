package handler

import (
	"database/sql"
	"log"
	"testing"

	"github.com/julianlk522/modeep/db"
	"github.com/julianlk522/modeep/dbtest"
)

const (
	TEST_LOGIN_NAME    = "jlk"
	TEST_USER_ID       = "3"
	TEST_REQ_USER_ID   = "13"
	TEST_LINK_ID       = "1"
)
var (
	TestClient *sql.DB
	
	test_single_cat    = []string{"umvc3"}
	test_multiple_cats = []string{"umvc3", "flowers"}
)

func TestMain(m *testing.M) {
	if err := dbtest.SetupTestDB(); err != nil {
		log.Fatal(err)
	}
	// TestClient unneeded but helps to reiterate in tests that the DB connection is temporary
	TestClient = db.Client
	m.Run()
}
