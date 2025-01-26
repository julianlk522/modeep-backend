package model

import "testing"

func TestContainsInvalidChars(t *testing.T) {
	var test_login_names = []struct {
		LoginName string
		Valid      bool
	}{
		{"alltext", true},
		{"text4ndNumb3r5", true},
		{"has_underscore", true},
		{"YELLINGVOICE", true},
		{"1234567890", true},
		{"has space", false},
		{"otherWeirdChars$%*", false},
		{";;;;", false},
		{"::::", false},
		{"hypen-also-unacceptable", false},
		{"~~~~", false},
		{"```", false},
		{"///", false},
		{"\\\\", false},
		{",,,,", false},
		{"...", false},
		{"????", false},
	}

	for _, l := range test_login_names {
		if got := ContainsInvalidChars(l.LoginName); l.Valid == got {
			t.Fatalf("expected %t for %s, got %t", l.Valid, l.LoginName, got)
		}
	}
}
