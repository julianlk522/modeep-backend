package handler

import (
	"image"

	_ "golang.org/x/image/webp"

	"os"
	"testing"

	e "github.com/julianlk522/fitm/error"
)

func TestLoginNameTaken(t *testing.T) {
	var test_login_names = []struct {
		login_name string
		Taken      bool
	}{
		{"akjlhsdflkjhhasdf", false},
		{"janedoe", false},
		{"jlk", true},
	}

	for _, l := range test_login_names {
		if got := LoginNameTaken(l.login_name); l.Taken != got {
			t.Fatalf("expected %t, got %t", l.Taken, got)
		}
	}
}

func TestAuthenticateUser(t *testing.T) {
	var test_logins = []struct {
		LoginName string
		Password  string
		Valid     bool
	}{
		{"jlk", "password", false},
		{"monkey", "monkey", true},
		{"monkey", "bananas", false},
	}

	for _, l := range test_logins {
		is_authenticated, err := AuthenticateUser(l.LoginName, l.Password)
		if l.Valid && !is_authenticated {
			t.Fatalf("expected login name %s to be authenticated", l.LoginName)
		} else if !l.Valid && is_authenticated {
			t.Fatalf("login name %s NOT authenticated, expected error", l.LoginName)
		} else if err != nil && err != e.ErrInvalidLogin && err != e.ErrInvalidPassword {
			t.Fatalf("user %s failed with error: %s", l.LoginName, err)
		}
	}
}

// UploadProfilePic
func TestHasAcceptableAspectRatio(t *testing.T) {
	var test_image_files = []struct {
		Name                     string
		HasAcceptableAspectRatio bool
	}{
		{"test1.webp", false},
		{"test2.webp", false},
		{"test3.webp", true},
	}

	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		t.Fatal("FITM_TEST_DATA_PATH not set")
	}
	pic_dir := test_data_path + "/img/profile"

	for _, l := range test_image_files {
		f, err := os.Open(pic_dir + "/" + l.Name)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Fatal(err)
		}

		if HasAcceptableAspectRatio(img) != l.HasAcceptableAspectRatio {
			t.Fatalf("expected image %s to be %t", l.Name, l.HasAcceptableAspectRatio)
		}
	}
}

// DeleteProfilePic
func TestUserWithIDHasProfilePic(t *testing.T) {
	var test_users = []struct {
		ID            string
		HasProfilePic bool
	}{
		// test user jlk has profile pic
		{test_user_id, true},
		// test user nelson does not have profile pic
		{"nelson", false},
	}

	for _, u := range test_users {
		if got := UserWithIDHasProfilePic(u.ID); u.HasProfilePic != got {
			t.Fatalf("expected %t, got %t", u.HasProfilePic, got)
		}
	}
}
