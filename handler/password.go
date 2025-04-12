package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/render"
	"golang.org/x/crypto/bcrypt"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/handler/util"
	"github.com/julianlk522/fitm/model"
)

func AttemptPasswordReset(w http.ResponseWriter, r *http.Request) {
	var login_name string = r.URL.Query().Get("login_name")
	if login_name == "" {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrNoLoginName))
		return
	}

	user_exists, err := util.UserExists(login_name)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	} else if !user_exists {
		// fail silently
		render.Status(r, http.StatusOK)
		return
	}

	email, err := util.GetEmailFromLoginName(login_name)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	} else if email == "" {
		// fail silently
		render.Status(r, http.StatusOK)
		return
	}

	if err = util.EmailPasswordResetLink(login_name, email); err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	new_password_data := &model.NewPasswordRequest{}
	if err := render.Bind(r, new_password_data); err != nil {
		render.Render(w, r, e.ErrUnprocessable(err))
		return
	}

	payload, err := util.ValidatePasswordResetToken(new_password_data.Token)
	if err != nil {
		if err == e.ErrInvalidTokenFormat || err == e.ErrInvalidTokenSignature {
			render.Render(w, r, e.ErrInvalidRequest(err))
			return
		} else if err == e.ErrTokenExpired {
			render.Render(w, r, e.ErrUnauthenticated(err))
			return
		} else {
			render.Render(w, r, e.Err500(err))
			return
		}
	}

	pw_hash, err := bcrypt.GenerateFromPassword(
		[]byte(new_password_data.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatal(err)
	}

	if _, err = db.Client.Exec(
		`UPDATE Users SET password = ? WHERE login_name = ?`,
		pw_hash,
		payload.LoginName,
	); err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusOK)
}
