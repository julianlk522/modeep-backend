package handler

import (
	"bytes"
	"encoding/json"

	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignUp(t *testing.T) {
	test_signup_requests := []struct {
		Payload map[string]string
		Valid   bool
	}{
		{
			Payload: map[string]string{
				"login_name": "",
				"password":   "testtest",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "p",
				"password":   "testtest",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "123456789012345678901234567890123",
				"password":   "testtest",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "test",
				"password":   "",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "test",
				"password":   "pp",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "test",
				"password":   "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "jlk",
				"password":   "testtest",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"login_name": "test",
				"password":   "testtest",
			},
			Valid: true,
		},
	}

	for _, tr := range test_signup_requests {
		pl, _ := json.Marshal(tr.Payload)
		r := httptest.NewRequest(
			http.MethodPost,
			"/signup",
			bytes.NewReader(pl),
		)
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		SignUp(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != 201 {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			} else {
				t.Fatalf(
					"expected status code 201, got %d (test request %+v)\n%s", res.StatusCode,
					tr.Payload,
					text,
				)
			}
		} else if !tr.Valid && res.StatusCode != 400 {
			t.Fatalf("expected status code 400, got %d", res.StatusCode)
		}
	}
}