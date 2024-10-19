package model

import "regexp"

func ContainsInvalidChars(login_name string) bool {
	return regexp.MustCompile(`\W`).MatchString(login_name)
}