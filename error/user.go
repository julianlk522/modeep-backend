package error

import (
	"errors"
	"fmt"
)

var (
	ErrNoLoginName                   error = errors.New("no name provided")
	ErrNoPassword                    error = errors.New("no password provided")
	ErrNoEmail                       error = errors.New("no email provided")
	ErrInvalidLogin                  error = errors.New("invalid name or password")
	ErrInvalidPassword               error = errors.New("invalid password")
	ErrLoginNameTaken                error = errors.New("login name taken")
	ErrLoginNameContainsInvalidChars error = errors.New("name contains invalid characters ([a-zA-Z0-9_] allowed)")
	ErrNoJWTSecretEnv                error = errors.New("MODEEP_JWT_SECRET env var not set")
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
