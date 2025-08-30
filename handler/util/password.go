package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
	gomail "gopkg.in/mail.v2"
)

const PW_RESET_TOKEN_VALID_DURATION = 10 * time.Minute

func GetEmailFromLoginName(loginName string) (string, error) {
	var email sql.NullString
	if err := db.Client.QueryRow("SELECT email FROM Users WHERE login_name = ?", loginName).Scan(&email); err != nil {
		return "", err
	}
	return email.String, nil
}

func GeneratePasswordResetToken(login_name, email string) (string, error) {
	payload := model.PasswordResetPayload{
		LoginName: login_name,
		Email:     email,
		ExpiresAt: time.Now().Add(PW_RESET_TOKEN_VALID_DURATION),
	}

	payload_bytes, err := json.Marshal(payload)
	if err != nil {
		return "", e.FailedToMarshalPayload(err)
	}
	encoded_payload := base64.URLEncoding.EncodeToString(payload_bytes)

	secret := os.Getenv("FITM_PW_RESET_SECRET")
	if secret == "" {
		return "", e.ErrNoPasswordResetSecretEnv
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encoded_payload))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	token := fmt.Sprintf("%s*%s", encoded_payload, signature)
	return token, nil
}

func EmailPasswordResetLink(login_name string, email string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "Modeep Notification <no-reply@modeep.org>")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Modeep Password Reset Request")

	token, err := GeneratePasswordResetToken(login_name, email)
	if err != nil {
		return err
	}
	reset_URL := "https://modeep.org/reset-password?token=" + token

	m.SetBody("text/plain", fmt.Sprintf("Someone, hopefully you, requested a password reset for %s on modeep.org. Your password has not yet changed. To change it, please go to %s. If you don't want to update your password, you can ignore this email.", login_name, reset_URL))

	d := gomail.NewDialer(
		os.Getenv("FITM_SMTP_HOST"),
		587,
		"no-reply@modeep.org",
		os.Getenv("FITM_SMTP_PASS"),
	)
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func ValidatePasswordResetToken(token string) (*model.PasswordResetPayload, error) {
	secret := os.Getenv("FITM_PW_RESET_SECRET")
	if secret == "" {
		return nil, e.ErrNoPasswordResetSecretEnv
	}

	token_parts := strings.Split(token, "*")
	if len(token_parts) != 2 {
		return nil, e.ErrInvalidTokenFormat
	}
	encoded_payload, signature := token_parts[0], token_parts[1]

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encoded_payload))
	expected_signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected_signature)) {
		return nil, e.ErrInvalidTokenSignature
	}

	payload, err := GetDecodedPayload(encoded_payload)
	if err != nil {
		return nil, err
	}

	if time.Now().After(payload.ExpiresAt) {
		return nil, e.ErrTokenExpired
	}

	return payload, nil
}

func GetDecodedPayload(encoded_payload string) (*model.PasswordResetPayload, error) {
	decoded_payload_bytes, err := base64.URLEncoding.DecodeString(encoded_payload)
	if err != nil {
		return nil, e.FailedToDecodePayload(err)
	}

	var payload model.PasswordResetPayload
	if err := json.Unmarshal(decoded_payload_bytes, &payload); err != nil {
		return nil, e.FailedToUnmarshalPayload(err)
	}

	return &payload, nil
}
