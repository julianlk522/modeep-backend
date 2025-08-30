package handler

import (
	"log"
	"testing"

	"github.com/julianlk522/modeep/dbtest"
)

func TestMain(m *testing.M) {
	err := dbtest.SetupTestDB()
	if err != nil {
		log.Fatal(err)
	}
	m.Run()
}

const (
	TEST_USER_ID    = "3"
	TEST_LOGIN_NAME = "jlk"
)
