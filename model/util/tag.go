package model

import (
	"slices"
	"strings"
)

func HasTooLongCats(cats string) bool {
	split_cats := strings.SplitSeq(cats, ",")

	for cat := range split_cats {
		if len(cat) > CAT_CHAR_LIMIT {
			return true
		}
	}

	return false
}

func HasTooManyCats(cats string) bool {
	return strings.Count(cats, ",") + 1 > CATS_PER_LINK_LIMIT
}

func HasDuplicateCats(cats string) bool {
	split_cats := strings.Split(cats, ",")

	var found_cats = []string{}

	for i := range split_cats {
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

	for i := range split_cats {
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
