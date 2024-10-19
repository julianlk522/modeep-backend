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
	LifeSpanOverlap float32
	Cats            string
}

type TagRankingPublic struct {
	TagRanking
	SubmittedBy string
	LastUpdated string
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

func (t *NewTagRequest) Bind(r *http.Request) error {
	if t.NewTag.LinkID == "" {
		return e.ErrNoLinkID
	}

	switch {
	case t.NewTag.Cats == "":
		return e.ErrNoCats
	case util.HasTooLongCats(t.NewTag.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(t.NewTag.Cats):
		return e.NumCatsExceedsLimit(util.NUM_CATS_LIMIT)
	case util.HasDuplicateCats(t.NewTag.Cats):
		return e.ErrDuplicateCats
	}

	// capitalize 'nsfw' if found
	t.NewTag.Cats = util.CapitalizeNSFWCatIfNotAlready(t.NewTag.Cats)

	t.ID = uuid.New().String()
	t.Cats = util.TrimExcessAndTrailingSpaces(t.NewTag.Cats)
	t.LastUpdated = util.NEW_LONG_TIMESTAMP()

	return nil
}

type EditTagRequest struct {
	ID          string `json:"tag_id"`
	Cats        string `json:"cats"`
	LastUpdated string
}

func (et *EditTagRequest) Bind(r *http.Request) error {
	if et.ID == "" {
		return e.ErrNoTagID
	}

	switch {
	case et.Cats == "":
		return e.ErrNoCats
	case util.HasTooLongCats(et.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(et.Cats):
		return e.NumCatsExceedsLimit(util.NUM_CATS_LIMIT)
	case util.HasDuplicateCats(et.Cats):
		return e.ErrDuplicateCats
	}

	// capitalize 'nsfw' if found
	et.Cats = util.CapitalizeNSFWCatIfNotAlready(et.Cats)

	et.Cats = util.TrimExcessAndTrailingSpaces(et.Cats)
	et.LastUpdated = util.NEW_LONG_TIMESTAMP()

	return nil
}

type DeleteTagRequest struct {
	ID string `json:"tag_id"`
}

func (dt *DeleteTagRequest) Bind(r *http.Request) error {
	if dt.ID == "" {
		return e.ErrNoTagID
	}

	return nil
}
