package model

import (
	"net/http"
	"strings"

	e "github.com/julianlk522/modeep/error"

	util "github.com/julianlk522/modeep/model/util"

	"github.com/google/uuid"
)

type HasCats interface {
	Link | LinkSignedIn

	GetCats() string
}

type Link struct {
	ID                 string
	URL                string
	SubmittedBy        string
	SubmitDate         string
	Cats               string
	Summary            string
	SummaryCount       int
	TimesStarred       int64
	AvgStars           float32
	EarliestStarrers   string
	ClickCount         int64
	TagCount           int
	PreviewImgFilename string
}

func (l Link) GetCats() string {
	return l.Cats
}

type LinkSignedIn struct {
	Link
	StarsAssigned uint8
}

func (lsi LinkSignedIn) GetCats() string {
	return lsi.Cats
}

type LinksPageOptions struct {
	Cats string
	NSFW bool
}

type LinksPage[T Link | LinkSignedIn] struct {
	Links          *[]T
	NSFWLinksCount int
	MergedCats     []string
	Pages          int
}

type Contributor struct {
	LoginName      string
	LinksSubmitted int
}

type NewLink struct {
	*NewLinkRequest
	LinkExtraMetadata
	SubmittedBy        string
	SummaryCount       int
	PreviewImgFilename string
}

type NewLinkRequest struct {
	URL        string
	Cats       string
	Summary    string
	LinkID     string `json:"ID"`
	SubmitDate string
}

func (nlr *NewLinkRequest) Bind(r *http.Request) error {
	if nlr.URL == "" {
		return e.ErrNoURL
	} else if len(nlr.URL) > util.URL_CHAR_LIMIT {
		return e.ErrLinkURLCharsExceedLimit(util.URL_CHAR_LIMIT)
	}

	switch {
	case nlr.Cats == "":
		return e.ErrNoTagCats
	case util.HasTooLongCats(nlr.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(nlr.Cats):
		return e.NumCatsExceedsLimit(util.CATS_PER_LINK_LIMIT)
	case util.HasDuplicateCats(nlr.Cats):
		return e.ErrDuplicateCats
	}

	if len(nlr.Summary) > util.SUMMARY_CHAR_LIMIT {
		return e.SummaryLengthExceedsLimit(util.SUMMARY_CHAR_LIMIT)
	}

	if strings.Contains(nlr.Summary, "\"") {
		nlr.Summary = strings.ReplaceAll(nlr.Summary, "\"", "'")
	}

	nlr.LinkID = uuid.New().String()
	nlr.SubmitDate = util.NEW_LONG_TIMESTAMP()

	return nil
}

type LinkExtraMetadata struct {
	AutoSummary   string
	PreviewImgURL string
}

type YTVideoMetadata struct {
	ID    string
	Items []YTVideoItems `json:"items"`
}

type YTVideoItems struct {
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

type DeleteLinkRequest struct {
	LinkID string `json:"link_id"`
}

func (dlr *DeleteLinkRequest) Bind(r *http.Request) error {
	if dlr.LinkID == "" {
		return e.ErrNoLinkID
	}

	return nil
}

type UnstarLinkRequest struct {
	LinkID string `json:"link_id"`
}

func (ulr *UnstarLinkRequest) Bind(r *http.Request) error {
	if ulr.LinkID == "" {
		return e.ErrNoLinkID
	}

	return nil
}

type StarLinkRequest struct {
	*UnstarLinkRequest
	Stars uint8 `json:"stars"`
}

func (sl *StarLinkRequest) Bind(r *http.Request) error {
	if sl.LinkID == "" {
		return e.ErrNoLinkID
	}

	if sl.Stars > 3 {
		return e.ErrInvalidStars
	}

	return nil
}

type NewClickRequest struct {
	LinkID    string `json:"link_id"`
	IPAddr    string
	Timestamp string
}

func (ncr *NewClickRequest) Bind(r *http.Request) error {
	if ncr.LinkID == "" {
		return e.ErrNoLinkID
	}

	ncr.Timestamp = util.NEW_LONG_TIMESTAMP()

	return nil
}
