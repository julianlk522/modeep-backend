package error

import (
	"errors"
	"fmt"
)

var (
	// Query links
	ErrInvalidLinkID       error = errors.New("invalid link ID provided")
	ErrInvalidPeriod       error = errors.New("invalid period provided")
	ErrInvalidPageParams   error = errors.New("invalid page provided")
	ErrInvalidNSFWParams   error = errors.New("invalid NSFW params provided")
	ErrInvalidSortByParams error = errors.New("invalid sort_by params provided")
	ErrInvalidStars        error = errors.New("invalid number of stars provided")
	ErrSameNumberOfStars   error = errors.New("invalid number of stars provided: same as before")
	ErrNoLinkID            error = errors.New("no link ID provided")
	ErrNoLinkWithID        error = errors.New("no link found with given ID")
	ErrNoCats              error = errors.New("no cats provided")
	ErrNoPeriod            error = errors.New("no period provided")
	// Preview Img
	ErrPreviewImgNotFound error = errors.New("preview image not found at specified path")
	// Add link
	ErrNoURL                 error = errors.New("no URL provided")
	ErrInvalidURL            error = errors.New("invalid URL provided")
	ErrGoogleAPIsKeyNotFound error = errors.New("gAPIs key not found")
	ErrRedirect              error = errors.New("invalid link destination: redirect detected")
	ErrCannotStarOwnLink     error = errors.New("cannot star your own link")
	ErrLinkAlreadyStarred     error = errors.New("link already starred")
	ErrLinkNotStarred         error = errors.New("link is not starred")
	// Delete link
	ErrDoesntOwnLink error = errors.New("not your link; cannot delete")
	// Click link
	ErrNoUserOrIP error = errors.New("click cannot be recorded without either authorized user ID or IP (neither found)")
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
