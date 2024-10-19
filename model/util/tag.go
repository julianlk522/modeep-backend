package model

import (
	"slices"
	"strings"
)

func HasTooLongCats(cats string) bool {
	split_cats := strings.Split(cats, ",")

	for _, cat := range split_cats {
		if len(cat) > CAT_CHAR_LIMIT {
			return true
		}
	}

	return false
}

func HasTooManyCats(cats string) bool {
	num_cats := strings.Count(cats, ",") + 1
	// +1 since "a" (no commas) would be one cat
	// and "a,b" (one comma) would be two
	return num_cats > NUM_CATS_LIMIT
}

func HasDuplicateCats(cats string) bool {
	split_cats := strings.Split(cats, ",")

	var found_cats = []string{}

	for i := 0; i < len(split_cats); i++ {
		if !slices.Contains(found_cats, split_cats[i]) {
			found_cats = append(found_cats, split_cats[i])
		} else {
			return true
		}
	}

	return false
}

func CapitalizeNSFWCatIfNotAlready(cats string) string {
	split_cats := strings.Split(cats, ",")

	for i := 0; i < len(split_cats); i++ {
		if split_cats[i] == "nsfw" {
			split_cats[i] = "NSFW"
		}
	}

	return strings.Join(split_cats, ",")
}

func TrimExcessAndTrailingSpaces(cats string) string {

	// remove leading / trailing spaces
	cats = strings.TrimSpace(cats)

	// replace multiple spaces with single
	cats = strings.Join(strings.Fields(cats), " ")

	return cats
}
