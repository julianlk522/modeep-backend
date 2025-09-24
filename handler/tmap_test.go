package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	m "github.com/julianlk522/modeep/middleware"
)

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
		jwt_claims := map[string]any{
			"user_id":    TEST_USER_ID,
			"login_name": TEST_LOGIN_NAME,
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		EditAbout(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != http.StatusOK {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			} else {
				t.Fatalf(
					"expected status code %d, got %d (test request %+v)\n%s", 
					res.StatusCode,
					http.StatusOK,
					tr.Payload,
					text,
				)
			}
		} else if !tr.Valid && res.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d (test request %+v)", 
			res.StatusCode, 
			http.StatusBadRequest,
			tr.Payload,
			)
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
			// test user jlk has a profile pic: should be able to delete it
			UserID:             TEST_USER_ID,
			Valid:              true,
			ExpectedStatusCode: http.StatusNoContent,
		},
		{
			// test user bradley (id 9) does not have a profile pic: should fail
			UserID:             "9",
			Valid:              false,
			ExpectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tr := range test_requests {
		r := httptest.NewRequest(
			http.MethodDelete,
			"/pic",
			nil,
		)

		ctx := context.Background()
		jwt_claims := map[string]any{
			"user_id": TEST_USER_ID,
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
			t.Fatalf("expected status code %d, got %d (test request %+v)", 
				tr.ExpectedStatusCode,
				res.StatusCode, 
				tr,
			)
		}
	}
}

func TestGetTreasureMap(t *testing.T) {
	// TODO
}
