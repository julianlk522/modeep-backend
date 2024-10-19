package error

import (
	"errors"
	"fmt"
)

var (
	ErrNoTagID             error = errors.New("no tag ID provided")
	ErrNoTagCats           error = errors.New("no tag cat(s) provided")
	ErrNoGlobalCatsSnippet error = errors.New("no global cats snippet provided")
	ErrNoOmittedCats       error = errors.New("no omitted cats provided")
	ErrNoTagWithID         error = errors.New("no tag found with given ID")
	ErrNoUserWithLoginName error = errors.New("no user found with given login name")
	ErrInvalidMoreFlag error = errors.New("invalid value passed as \"more\" params. should be unset or \"true\"")
	ErrDuplicateTag        error = errors.New("duplicate tag")
	ErrDuplicateCats       error = errors.New("tag contains duplicate cat(s)")
	ErrDoesntOwnTag        error = errors.New("not your tag")
	ErrCantDeleteOnlyTag   error = errors.New("last tag for this link; cannot be deleted")
)

func CatCharsExceedLimit(limit int) error {
	return fmt.Errorf("cat too long (max %d chars)", limit)
}

func NumCatsExceedsLimit(limit int) error {
	return fmt.Errorf("too many tag cats (%d max)", limit)
}
