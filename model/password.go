package model

import (
	"net/http"
	"time"

	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/model/util"
)

type PasswordResetPayload struct {
	LoginName string    `json:"login_name"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"exp"`
}

type NewPasswordRequest struct {
	NewPassword string `json:"new_password"`
	Token       string `json:"token"`
}

func (npr *NewPasswordRequest) Bind(r *http.Request) error {
	switch {
	case npr.NewPassword == "":
		return e.ErrNoPassword
	case npr.Token == "":
		return e.ErrNoPasswordResetToken
	case len(npr.NewPassword) < util.PASSWORD_LOWER_CHAR_LIMIT:
		return e.PasswordExceedsLowerLimit(util.PASSWORD_LOWER_CHAR_LIMIT)
	case len(npr.NewPassword) > util.PASSWORD_UPPER_CHAR_LIMIT:
		return e.PasswordExceedsUpperLimit(util.PASSWORD_UPPER_CHAR_LIMIT)
	default:
		return nil
	}
}
