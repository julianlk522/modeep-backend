package error

import (
	"errors"
	"fmt"
)

var (
	// Auth
	ErrInvalidLogin      error = errors.New("invalid login provided")
	ErrIncorrectPassword       = errors.New("incorrect password")
	ErrNoJWTSecretEnv          = errors.New("FITM_JWT_SECRET env var not set")
	ErrNoLoginName       error = errors.New("no name provided")
	ErrLoginNameContainsInvalidChars error = errors.New("name contains invalid characters ([a-zA-Z0-9_] allowed)")
	ErrLoginNameTaken    error = errors.New("login name taken")
	ErrNoPassword        error = errors.New("no password provided")
	// Tmap profile
	ErrAboutHasInvalidChars         error = errors.New("be more descriptive. (not just \\n or \\r)")
	ErrProfilePicNotFound           error = errors.New("profile pic not found")
	ErrInvalidFileType              error = errors.New("invalid file provided (accepted image formats: .jpg, .jpeg, .png, .webp)")
	ErrInvalidProfilePicAspectRatio error = errors.New("profile pic aspect ratio must be no more than 2:1 and no less than 0.5:1")
	ErrCouldNotCreateProfilePic     error = errors.New("could not create new profile pic file")
	ErrCouldNotCopyProfilePic       error = errors.New("could not save profile pic to file")
	ErrCouldNotSaveProfilePic       error = errors.New("could not assign profile pic to user")
	ErrCouldNotRemoveProfilePic     error = errors.New("could not remove profile pic for user")
	ErrNoProfilePic                 error = errors.New("no profile pic found for user")
)

func LoginNameExceedsLowerLimit(limit int) error {
	return fmt.Errorf("name too short (min %d chars)", limit)
}

func LoginNameExceedsUpperLimit(limit int) error {
	return fmt.Errorf("name too long (max %d chars)", limit)
}

func PasswordExceedsLowerLimit(limit int) error {
	return fmt.Errorf("password too short (min %d chars)", limit)
}

func PasswordExceedsUpperLimit(limit int) error {
	return fmt.Errorf("password too long (max %d chars)", limit)
}

func ProfileAboutLengthExceedsLimit(limit int) error {
	return fmt.Errorf("about text too long (max %d chars)", limit)
}
