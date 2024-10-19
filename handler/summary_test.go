package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	m "github.com/julianlk522/fitm/middleware"
)

func TestAddSummary(t *testing.T) {
	test_summary_requests := []struct {
		Payload map[string]string
		Valid   bool
	}{
		{
			Payload: map[string]string{
				"link_id": "",
				"text":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "-1",
				"text":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "notanint",
				"text":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "1",
				"text":    "",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "1",
				"text":    "too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text too much text",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "101010101010101010101010101010101010101",
				"text":    "testtest",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "1",
				"text":    "testtest",
			},
			Valid: true,
		},
	}

	const (
		test_user_id    = "3"
		test_login_name = "jlk"
	)

	for _, tr := range test_summary_requests {
		pl, _ := json.Marshal(tr.Payload)
		r := httptest.NewRequest(
			http.MethodPost,
			"/summaries",
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
		AddSummary(w, r)
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
