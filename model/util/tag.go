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
	return strings.Count(cats, ",") + 1 > NUM_CATS_LIMIT
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
	cats = strings.Join(strings.Fields(strings.TrimSpace(cats)), " ")

	return cats
}
