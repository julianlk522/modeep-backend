package model

import (
	"net/http"
	"strings"

	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/model/util"

	"github.com/google/uuid"
)

type Summary struct {
	ID             string
	Text           string
	SubmittedBy    string
	LastUpdated    string
	LikeCount      int
	EarliestLikers string
}

type SummarySignedIn struct {
	Summary
	IsLiked bool
}

type SummaryPage[S SummarySignedIn | Summary, L LinkSignedIn | Link] struct {
	Link      L
	Summaries []S
}

type NewSummaryRequest struct {
	ID          string
	LinkID      string `json:"link_id"`
	Text        string `json:"text"`
	LastUpdated string
}

func (nsr *NewSummaryRequest) Bind(r *http.Request) error {
	if nsr.LinkID == "" {
		return e.ErrNoLinkID
	}

	if nsr.Text == "" {
		return e.ErrNoSummaryText
	} else if len(nsr.Text) > util.SUMMARY_CHAR_LIMIT {
		return e.SummaryLengthExceedsLimit(util.SUMMARY_CHAR_LIMIT)
	}

	if strings.Contains(nsr.Text, "\"") {
		nsr.Text = strings.ReplaceAll(nsr.Text, "\"", "'")
	}

	nsr.ID = uuid.New().String()
	nsr.LastUpdated = util.NEW_LONG_TIMESTAMP()

	return nil

}

type DeleteSummaryRequest struct {
	SummaryID string `json:"summary_id"`
}

func (dsr *DeleteSummaryRequest) Bind(r *http.Request) error {
	if dsr.SummaryID == "" {
		return e.ErrNoSummaryID
	}
	return nil
}

type EditSummaryRequest struct {
	SummaryID string `json:"summary_id"`
	Text      string `json:"text"`
}

func (esr *EditSummaryRequest) Bind(r *http.Request) error {
	if esr.SummaryID == "" {
		return e.ErrNoSummaryID
	}
	if esr.Text == "" {
		return e.ErrNoSummaryReplacementText
	} else if len(esr.Text) > util.SUMMARY_CHAR_LIMIT {
		return e.SummaryLengthExceedsLimit(util.SUMMARY_CHAR_LIMIT)
	}

	if strings.Contains(esr.Text, "\"") {
		esr.Text = strings.ReplaceAll(esr.Text, "\"", "'")
	}

	return nil
}
