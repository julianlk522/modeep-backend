package model

import (
	"net/http"
	"strings"

	e "github.com/julianlk522/modeep/error"
	util "github.com/julianlk522/modeep/model/util"

	"github.com/google/uuid"
)

// TAGS
type Tag struct {
	ID          string
	LinkID      string
	Cats        string
	SubmittedBy string
	LastUpdated string
}

type TagPage[T Link | LinkSignedIn] struct {
	Link        *T
	YourTag     *Tag
	TagRankings *[]TagRanking
}

type TagRanking struct {
	LifeSpanOverlap float32
	Cats            string
	SubmittedBy     string
	LastUpdated     string
}

// INDIVIDUAL CATS
type CatCount struct {
	Category string
	Count    int32
}

type TopCatCountsOptions struct {
	RawCatFilters      []string
	NeuteredCatFilters []string
	SummaryContains    string
	URLContains        string
	URLLacks           string
	Period             Period
	More               bool
}

func SortCats(i, j CatCount) int {
	if i.Count > j.Count {
		return -1
	} else if i.Count == j.Count && strings.ToLower(i.Category) < strings.ToLower(j.Category) {
		return -1
	}
	return 1
}

// SPELLFIX
type SpellfixMatchesOptions struct {
	IsTmapAndOwnerIs string
	CatFilters       []string
	YouAreAddingCats bool
}

// for CalculateAndSetGlobalCats()
type CatRanking struct {
	Cat   string
	Score float32
}

type GlobalCatsDiff struct {
	Added   []string
	Removed []string
}

// REQUESTS
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
		return e.NumCatsExceedsLimit(util.CATS_PER_LINK_LIMIT)
	case util.HasDuplicateCats(ntr.NewTag.Cats):
		return e.ErrDuplicateCats
	}

	ntr.ID = uuid.New().String()
	ntr.NewTag.Cats = util.CapitalizeNSFWCatIfNotAlready(ntr.NewTag.Cats)
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
		return e.NumCatsExceedsLimit(util.CATS_PER_LINK_LIMIT)
	case util.HasDuplicateCats(etr.Cats):
		return e.ErrDuplicateCats
	}

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
