package model

import "testing"

func TestContainsInvalidChars(t *testing.T) {
	var test_login_names = []struct {
		login_name string
		valid bool
	} {
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
		return_true := ContainsInvalidChars(l.login_name)
		if l.valid && return_true {
			t.Fatalf("expected login name %s to be valid", l.login_name)
		} else if !l.valid && !return_true {
			t.Fatalf("login name %s NOT valid, expected error", l.login_name)
		}
	}
}