package handler

import (
	"database/sql"
	"log"
	"testing"

	"github.com/julianlk522/fitm/db"
	"github.com/julianlk522/fitm/dbtest"
)

// shared across handler/util tests
var (
	TestClient *sql.DB

	test_login_name = "jlk"
	test_user_id    = "3"

	test_req_user_id    = "13"

	test_single_cat    = []string{"umvc3"}
	test_multiple_cats = []string{"umvc3", "flowers"}

	test_link_id = "1"
)

func TestMain(m *testing.M) {
	if err := dbtest.SetupTestDB(); err != nil {
		log.Fatal(err)
	}
	// TestClient unneeded but helps to reiterate in tests that the DB connection is temporary
	TestClient = db.Client
	m.Run()
}
