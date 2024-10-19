package error

import (
	"errors"
	"fmt"
)

var (
	ErrNoSummaryID              error = errors.New("no summary ID provided")
	ErrNoSummaryWithID          error = errors.New("no summary found with given ID")
	ErrNoSummaryText            error = errors.New("no summary text provided")
	ErrNoSummaryReplacementText error = errors.New("no summary replacement text provided")
	ErrDoesntOwnSummary         error = errors.New("not your summary")
	ErrCannotLikeOwnSummary     error = errors.New("cannot like your own summary")
	ErrSummaryAlreadyLiked      error = errors.New("summary already liked")
	ErrSummaryNotLiked          error = errors.New("summary not already liked")
)

func SummaryLengthExceedsLimit(limit int) error {
	return fmt.Errorf("summary too long (max %d chars)", limit)
}
