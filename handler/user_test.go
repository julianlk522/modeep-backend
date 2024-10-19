package handler

import (
	"bytes"
	"context"
	"encoding/json"

	m "github.com/julianlk522/fitm/middleware"

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

func TestEditAbout(t *testing.T) {
	test_edit_about_requests := []struct {
		Payload map[string]string
		Valid   bool
	}{
		{
			Payload: map[string]string{
				"about": "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"about": "hello",
			},
			Valid: true,
		},
		{
			Payload: map[string]string{
				"about": "",
			},
			Valid: true,
		},
		// not allowed: must have chars if not empty
		{
			Payload: map[string]string{
				"about": "\n\r",
			},
			Valid: false,
		},
	}

	for _, tr := range test_edit_about_requests {
		pl, _ := json.Marshal(tr.Payload)
		r := httptest.NewRequest(
			http.MethodPut,
			"/about",
			bytes.NewReader(pl),
		)
		r.Header.Set("Content-Type", "application/json")

		ctx := context.Background()
		jwt_claims := map[string]interface{}{
			"user_id":    test_user_id,
			"login_name": test_login_name,
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		EditAbout(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != 200 {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			} else {
				t.Fatalf(
					"expected status code 200, got %d (test request %+v)\n%s", res.StatusCode,
					tr.Payload,
					text,
				)
			}
		} else if !tr.Valid && res.StatusCode != 400 {
			t.Fatalf("expected status code 400, got %d (test request %+v)", res.StatusCode, tr.Payload)
		}
	}
}

func TestDeleteProfilePic(t *testing.T) {
	var test_requests = []struct {
		UserID             string
		Valid              bool
		ExpectedStatusCode int
	}{
		{
			// jlk has a profile pic: should be able to delete it
			UserID:             test_user_id,
			Valid:              true,
			ExpectedStatusCode: 204,
		},
		{
			// bradley does not have a profile pic: should fail
			UserID:             "9",
			Valid:              false,
			ExpectedStatusCode: 400,
		},
	}

	for _, tr := range test_requests {
		r := httptest.NewRequest(
			http.MethodDelete,
			"/pic",
			nil,
		)

		ctx := context.Background()
		jwt_claims := map[string]interface{}{
			"user_id": test_user_id,
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		DeleteProfilePic(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != tr.ExpectedStatusCode {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			} else {
				t.Fatalf(
					"expected status code %d, got %d (test request %+v)\n%s", tr.ExpectedStatusCode, res.StatusCode,
					tr,
					text,
				)
			}
		} else if !tr.Valid && res.StatusCode != tr.ExpectedStatusCode {
			t.Fatalf("expected status code 400, got %d (test request %+v)", res.StatusCode, tr)
		}
	}
}

func TestGetTreasureMap(t *testing.T) {
	// TODO
}
