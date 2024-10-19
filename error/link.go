package error

import (
	"errors"
	"fmt"
)

var (
	// Query links
	ErrInvalidPage       error = errors.New("invalid page provided")
	ErrInvalidLinkID     error = errors.New("invalid link ID provided")
	ErrInvalidPeriod     error = errors.New("invalid period provided")
	ErrInvalidNSFWParams error = errors.New("invalid NSFW params provided")
	ErrNoLinkID          error = errors.New("no link ID provided")
	ErrNoLinkWithID      error = errors.New("no link found with given ID")
	ErrNoCats            error = errors.New("no cats provided")
	ErrNoPeriod          error = errors.New("no period provided")
	// Add link
	ErrNoURL                 error = errors.New("no URL provided")
	ErrInvalidURL            error = errors.New("invalid URL provided")
	ErrGoogleAPIsKeyNotFound error = errors.New("gAPIs key not found")
	ErrRedirect              error = errors.New("invalid link destination: redirect detected")
	ErrCannotLikeOwnLink     error = errors.New("cannot like your own link")
	ErrLinkAlreadyLiked      error = errors.New("link already liked")
	ErrLinkNotLiked          error = errors.New("link not already liked")
	ErrCannotCopyOwnLink     error = errors.New("cannot copy your own link to your treasure map")
	ErrLinkAlreadyCopied     error = errors.New("link already copied to treasure map")
	ErrLinkNotCopied         error = errors.New("link not already copied")
	// Delete link
	ErrDoesntOwnLink error = errors.New("not your link; cannot delete")
)

func ErrMaxDailyLinkSubmissionsReached(limit int) error {
	return fmt.Errorf("you have submitted the max amount of links for today (%d)", limit)
}

func ErrLinkURLCharsExceedLimit(limit int) error {
	return fmt.Errorf("URL too long (max %d chars)", limit)
}

func ErrGoogleAPIsRequestFail(err error) error {
	return fmt.Errorf("error requesting from GoogleAPIs: %s", err)
}

func ErrInvalidGoogleAPIsResponse(status_text string) error {
	return fmt.Errorf("invalid response from GoogleAPIs: %s", status_text)
}

func ErrGoogleAPIsResponseExtractionFail(err error) error {
	return fmt.Errorf("error extracting response from GoogleAPIs: %s", err)
}

func ErrDuplicateLink(url string, duplicate_link_id string) error {
	return fmt.Errorf(
		"URL %s already submitted. See /tag/%s",
		url,
		duplicate_link_id,
	)
}
