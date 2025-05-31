package error

import (
	"errors"
	"fmt"
)

var (
	// Profile
	ErrAboutHasInvalidChars         error = errors.New("be more descriptive. (not just \\n or \\r)")
	ErrProfilePicNotFound           error = errors.New("profile pic not found")
	ErrInvalidFileType              error = errors.New("invalid file provided (accepted image formats: .jpg, .jpeg, .png, .webp)")
	ErrInvalidProfilePicAspectRatio error = errors.New("profile pic aspect ratio must be no more than 2:1 and no less than 0.5:1")
	ErrCouldNotCreateProfilePicFile     error = errors.New("could not create new profile pic file")
	ErrCouldNotSaveResizedProfilePic       error = errors.New("could not save resized profile pic to file")
	ErrCouldNotDeleteProfilePicFile     error = errors.New("could not delete profile pic file")
	ErrNoProfilePic                 error = errors.New("no profile pic found for user")

	// Links
	ErrNoTmapOwnerLoginName error = errors.New("no login name provided for Treasure Map owner")
	ErrInvalidSectionParams error = errors.New("invalid section params provided")
	ErrInvalidOnlySectionParams error = errors.New("invalid params provided for single Treasure Map section")
)

func ProfileAboutLengthExceedsLimit(limit int) error {
	return fmt.Errorf("about text too long (max %d chars)", limit)
}
