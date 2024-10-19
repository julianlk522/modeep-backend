package model

import (
	"testing"
)

func TestCapitalizeNSFWCatIfNotAlready(t *testing.T) {
	var test_cats = []struct {
		Cats string
		Want string
	}{
		{"nsfw", "NSFW"},
		{"not,present", "not,present"},
		{"some,cats,first,nsfw", "some,cats,first,NSFW"},
	}

	for _, tc := range test_cats {
		got := CapitalizeNSFWCatIfNotAlready(tc.Cats)
		if got != tc.Want {
			t.Fatalf("got %s, want %s", got, tc.Want)
		}
	}
}

func TestTrimExcessAndTrailingSpaces(t *testing.T) {
	var test_cats = []struct {
		Cats string
		Want string
	}{
		{"hello     mom", "hello mom"},

		{"     hello mom", "hello mom"},

		{"hello     mom     ", "hello mom"},
		{"mikey", "mikey"},
		{"test name", "test name"},
	}

	for _, tc := range test_cats {
		got := TrimExcessAndTrailingSpaces(tc.Cats)
		if got != tc.Want {
			t.Fatalf("got %s, want %s", got, tc.Want)
		}
	}
}
