package model

import (
	"net/http"
	"strings"

	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/model/util"

	"github.com/google/uuid"
)

type Tag struct {
	ID          string
	LinkID      string
	Cats        string
	SubmittedBy string
	LastUpdated string
}

type CatCount struct {
	Category string
	Count    int32
}

func SortCats(i, j CatCount) int {
	if i.Count > j.Count {
		return -1
	} else if i.Count == j.Count && strings.ToLower(i.Category) < strings.ToLower(j.Category) {
		return -1
	}
	return 1
}

type TagRanking struct {
	Cats  string
	LifeSpanOverlap float32
}

type TagRankingPublic struct {
	TagRanking
	SubmittedBy string
	LastUpdated string
}

type CatRanking struct {
	Cat string
	Score float32
}

type GlobalCatsDiff struct {
	Added []string
	Removed []string
}

type TagPage[T Link | LinkSignedIn] struct {
	Link        *T
	UserTag     *Tag
	TagRankings *[]TagRankingPublic
}

type NewTag struct {
	LinkID string `json:"link_id"`
	Cats   string `json:"cats"`
}

type NewTagRequest struct {
	*NewTag
	ID          string
	LastUpdated string
}

func (ntr *NewTagRequest) Bind(r *http.Request) error {
	if ntr.NewTag.LinkID == "" {
		return e.ErrNoLinkID
	}

	switch {
	case ntr.NewTag.Cats == "":
		return e.ErrNoCats
	case util.HasTooLongCats(ntr.NewTag.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(ntr.NewTag.Cats):
		return e.NumCatsExceedsLimit(util.NUM_CATS_LIMIT)
	case util.HasDuplicateCats(ntr.NewTag.Cats):
		return e.ErrDuplicateCats
	}

	// capitalize 'nsfw' if found
	ntr.NewTag.Cats = util.CapitalizeNSFWCatIfNotAlready(ntr.NewTag.Cats)

	ntr.ID = uuid.New().String()
	ntr.Cats = util.TrimExcessAndTrailingSpaces(ntr.NewTag.Cats)
	ntr.LastUpdated = util.NEW_LONG_TIMESTAMP()

	return nil
}

type EditTagRequest struct {
	ID          string `json:"tag_id"`
	Cats        string `json:"cats"`
	LastUpdated string
}

func (etr *EditTagRequest) Bind(r *http.Request) error {
	if etr.ID == "" {
		return e.ErrNoTagID
	}

	switch {
	case etr.Cats == "":
		return e.ErrNoCats
	case util.HasTooLongCats(etr.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(etr.Cats):
		return e.NumCatsExceedsLimit(util.NUM_CATS_LIMIT)
	case util.HasDuplicateCats(etr.Cats):
		return e.ErrDuplicateCats
	}

	// capitalize 'nsfw' if found
	etr.Cats = util.CapitalizeNSFWCatIfNotAlready(etr.Cats)

	etr.Cats = util.TrimExcessAndTrailingSpaces(etr.Cats)
	etr.LastUpdated = util.NEW_LONG_TIMESTAMP()

	return nil
}

type DeleteTagRequest struct {
	ID string `json:"tag_id"`
}

func (dtr *DeleteTagRequest) Bind(r *http.Request) error {
	if dtr.ID == "" {
		return e.ErrNoTagID
	}

	return nil
}
