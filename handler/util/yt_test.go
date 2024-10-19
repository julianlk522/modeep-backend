package handler

import (
	"testing"
)

func TestIsYouTubeVideoLink(t *testing.T) {
	var test_urls = []struct {
		URL   string
		Valid bool
	}{
		{"https://www.youtube.com/watch?v=9bZkp7q19f0", true},
		{"https://www.youtube.com/watch?v=9bZkp7q19f0&feature=player_embedded", true},
		{"fred.com", false},
		{"https://www.youtube.com/watch?v=MH03ZJaNe8A", true},
		{"https://youtu.be/uW5GjbidEHU?si=d2wJ7ADMCMMyJfQ-", true},
		{"https://youtu.be/uW5GjbidEHU", true},
		{"https://web.archive.org/web/20050428014715/youtube.com/", false},
		{"https://web.archive.org/web/20240000000000*/https://www.youtube.com/watch?v=9bZkp7q19f0", false},
		{"test/youtube.com/watch?v=abcdefg", false},
	}

	for _, u := range test_urls {
		return_true := IsYouTubeVideoLink(u.URL)
		if u.Valid && !return_true {
			t.Fatalf("expected url %s to be valid", u.URL)
		} else if !u.Valid && return_true {
			t.Fatalf("url %s NOT valid, expected error", u.URL)
		}
	}
}

func TestObtainYouTubeMetaData(t *testing.T) {
	// TODO
}

func TestExtractYouTubeVideoID(t *testing.T) {
	var test_urls = []struct {
		URL string
		ID  string
	}{
		{"https://www.youtube.com/watch?v=9bZkp7q19f0", "9bZkp7q19f0"},
		{"https://www.youtube.com/watch?v=9bZkp7q19f0&feature=player_embedded", "9bZkp7q19f0"},
		{"https://www.youtube.com/watch?v=MH03ZJaNe8A", "MH03ZJaNe8A"},
		{"https://youtu.be/uW5GjbidEHU?si=d2wJ7ADMCMMyJfQ-", "uW5GjbidEHU"},
		{"https://youtu.be/uW5GjbidEHU", "uW5GjbidEHU"},
	}

	for _, u := range test_urls {
		id := ExtractYouTubeVideoID(u.URL)
		if id != u.ID {
			t.Fatalf("expected %s, got %s", u.ID, id)
		}
	}
}

func TestExtractMetaDataFromGoogleAPIsResponse(t *testing.T) {
	// TODO
}
