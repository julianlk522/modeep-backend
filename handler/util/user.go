package handler

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/go-chi/jwtauth/v5"
	"github.com/go-chi/render"
	_ "golang.org/x/image/webp"

	"time"

	"golang.org/x/crypto/bcrypt"
)

// Auth
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
		return false, e.ErrIncorrectPassword
	}

	return true, nil
}

func GetJWTFromLoginName(login_name string) (string, error) {
	var id sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Users WHERE login_name = ?", login_name).Scan(&id)
	if err != nil {
		return "", err
	}

	claims := map[string]interface{}{
		"user_id":    id.String,
		"login_name": login_name,
	}
	// TEST
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiry(claims, time.Now().Add(4*time.Hour))

	secret := os.Getenv("FITM_JWT_SECRET")
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

// Upload profile pic
func HasAcceptableAspectRatio(img image.Image) bool {
	b := img.Bounds()
	width, height := b.Max.X, b.Max.Y
	ratio := float64(width) / float64(height)

	if ratio > 2.0 || ratio < 0.5 {
		return false
	}

	return true
}

// Delete profile pic
func UserWithIDHasProfilePic(user_id string) bool {
	var p sql.NullString
	if err := db.Client.QueryRow("SELECT pfp FROM Users WHERE id = ?", user_id).Scan(&p); err != nil {
		return false
	}
	return p.Valid
}
