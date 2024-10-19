package model

import (
	"net/http"
	"strings"

	e "github.com/julianlk522/fitm/error"

	util "github.com/julianlk522/fitm/model/util"

	"github.com/google/uuid"
)

type Link struct {
	ID           string
	URL          string
	SubmittedBy  string
	SubmitDate   string
	Cats         string
	Summary      string
	SummaryCount int
	TagCount     int
	LikeCount    int64
	ImgURL       string
}

// YouTube links
type YTVideoMetaData struct {
	Items []VYTVideoItems `json:"items"`
}

type VYTVideoItems struct {
	Snippet YTVideoSnippet `json:"snippet"`
}

type YTVideoSnippet struct {
	Title      string `json:"title"`
	Thumbnails struct {
		Default struct {
			URL string `json:"url"`
		} `json:"default"`
	}
}

type LinkSignedIn struct {
	Link
	IsLiked  bool
	IsCopied bool
}

type PaginatedLinks[T Link | LinkSignedIn] struct {
	Links    *[]T
	NextPage int
}

type TmapLink struct {
	Link
	CatsFromUser bool
}

type TmapLinkSignedIn struct {
	LinkSignedIn
	CatsFromUser bool
}

type Contributor struct {
	LoginName      string
	LinksSubmitted int
}

type NewLink struct {
	URL     string `json:"url"`
	Cats    string `json:"cats"`
	Summary string `json:"summary,omitempty"`
}

type NewLinkRequest struct {
	*NewLink
	ID         string
	SubmitDate string
	LikeCount  int64

	// to be assigned by handler
	URL          string // potentially modified after test request(s)
	SubmittedBy  string
	Cats         string // potentially modified after sort
	AutoSummary  string
	SummaryCount int
	ImgURL       string
}

func (l *NewLinkRequest) Bind(r *http.Request) error {

	// URL
	if l.NewLink.URL == "" {
		return e.ErrNoURL
	} else if len(l.NewLink.URL) > util.URL_CHAR_LIMIT {
		return e.ErrLinkURLCharsExceedLimit(util.URL_CHAR_LIMIT)
	}

	// Cats
	switch {
	case l.NewLink.Cats == "":
		return e.ErrNoTagCats
	case util.HasTooLongCats(l.NewLink.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(l.NewLink.Cats):
		return e.NumCatsExceedsLimit(util.NUM_CATS_LIMIT)
	case util.HasDuplicateCats(l.NewLink.Cats):
		return e.ErrDuplicateCats
	}

	// Summary
	if len(l.NewLink.Summary) > util.SUMMARY_CHAR_LIMIT {
		return e.SummaryLengthExceedsLimit(util.SUMMARY_CHAR_LIMIT)
	}

	if strings.Contains(l.NewLink.Summary, "\"") {
		l.NewLink.Summary = strings.ReplaceAll(l.NewLink.Summary, "\"", "'")
	}

	l.ID = uuid.New().String()
	l.SubmitDate = util.NEW_LONG_TIMESTAMP()
	l.LikeCount = 0

	return nil
}

type DeleteLinkRequest struct {
	LinkID string `json:"link_id"`
}

func (dl *DeleteLinkRequest) Bind(r *http.Request) error {
	if dl.LinkID == "" {
		return e.ErrNoLinkID
	}

	return nil
}
