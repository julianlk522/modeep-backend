package handler

import (
	"log"
	"testing"

	"github.com/julianlk522/fitm/dbtest"
)

func TestMain(m *testing.M) {
	err := dbtest.SetupTestDB()
	if err != nil {
		log.Fatal(err)
	}
	m.Run()
}

const (
	test_user_id    = "3"
	test_login_name = "jlk"
)
