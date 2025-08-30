package handler

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/julianlk522/modeep/db"
	e "github.com/julianlk522/modeep/error"

	"github.com/go-chi/jwtauth/v5"
	"github.com/go-chi/render"
	_ "golang.org/x/image/webp"

	"time"

	"golang.org/x/crypto/bcrypt"
)

// Auth
func UserExists(login_name string) (bool, error) {
	var u sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Users WHERE login_name = ?;", login_name).Scan(&u)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func LoginNameTaken(login_name string) bool {
	var s sql.NullString
	if err := db.Client.QueryRow("SELECT login_name FROM Users WHERE login_name = ?", login_name).Scan(&s); err == nil {
		return true
	}
	return false
}

func AuthenticateUser(login_name string, password string) (bool, error) {
	var id, p sql.NullString
	if err := db.Client.QueryRow("SELECT id, password FROM Users WHERE login_name = ?", login_name).Scan(&id, &p); err != nil {
		if err == sql.ErrNoRows {
			return false, e.ErrInvalidLogin
		}
		return false, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(p.String), []byte(password)); err != nil {
		return false, e.ErrInvalidPassword
	}

	return true, nil
}

func GetJWTFromLoginName(login_name string) (string, error) {
	var id sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Users WHERE login_name = ?", login_name).Scan(&id)
	if err != nil {
		return "", err
	}

	claims := map[string]any{
		"user_id":    id.String,
		"login_name": login_name,
	}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiry(claims, time.Now().Add(4*time.Hour))

	secret := os.Getenv("MODEEP_JWT_SECRET")
	if secret == "" {
		return "", e.ErrNoJWTSecretEnv
	}
	auth := jwtauth.New(
		"HS256",
		[]byte(secret),
		nil,
	)
	_, token, err := auth.Encode(claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

func RenderJWT(token string, w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{"token": token})
}