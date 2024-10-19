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

func TestGetLinks(t *testing.T) {

	test_get_links_requests := []struct {
		Params map[string]string
		Page   int
		Valid  bool
	}{
		{
			Params: map[string]string{},
			Page:   0,
			Valid:  true,
		},
		{
			Params: map[string]string{},
			Page:   1,
			Valid:  true,
		},
		{
			Params: map[string]string{"cats": "umvc3"},
			Page:   1,
			Valid:  true,
		},
		{
			Params: map[string]string{
				"cats":   "umvc3",
				"period": "day",
			},
			Page:  1,
			Valid: true,
		},
		{
			Params: map[string]string{
				"cats":    "umvc3",
				"period":  "week",
				"sort_by": "newest",
			},
			Page:  1,
			Valid: true,
		},
		{
			Params: map[string]string{
				"cats":    "umvc3",
				"period":  "month",
				"sort_by": "rating",
			},
			Page:  1,
			Valid: true,
		},
		{
			Params: map[string]string{
				"cats":   "umvc3",
				"period": "poop",
			},
			Page:  1,
			Valid: false,
		},
		{
			Params: map[string]string{
				"req_user_id":    "3",
				"req_login_name": "jlk",
			},
			Page:  1,
			Valid: true,
		},
		// passes because middlware corrects negative pages to 1
		{
			Params: map[string]string{},
			Page:   -1,
			Valid:  true,
		},
		// fails: sort_by must be either "rating" or "newest"
		{
			Params: map[string]string{"sort_by": "invalid"},
			Page:   1,
			Valid:  false,
		},
		// nsfw params may be "true", "false", or absent but not anything else
		{
			Params: map[string]string{"nsfw": "true"},
			Page:   1,
			Valid:  true,
		},
		{
			Params: map[string]string{"nsfw": "false"},
			Page:   1,
			Valid:  true,
		},
		{
			Params: map[string]string{"nsfw": "invalid"},
			Page:   1,
			Valid:  false,
		},
		// NSFW in caps also valid
		{
			Params: map[string]string{"NSFW": "true"},
			Page:   1,
			Valid:  true,
		},
	}

	for _, tglr := range test_get_links_requests {
		r := httptest.NewRequest(
			http.MethodGet,
			"/links",
			nil,
		)

		ctx := context.Background()
		jwt_claims := map[string]interface{}{
			"user_id":    tglr.Params["req_user_id"],
			"login_name": tglr.Params["req_login_name"],
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		ctx = context.WithValue(ctx, m.PageKey, tglr.Page)
		r = r.WithContext(ctx)

		q := r.URL.Query()
		for k, v := range tglr.Params {
			q.Add(k, v)
		}
		r.URL.RawQuery = q.Encode()
				
		w := httptest.NewRecorder()
		GetLinks(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tglr.Valid && res.StatusCode != http.StatusOK {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			}

			t.Fatalf(
				"expected status code 200, got %d (test request %+v)\n%s", res.StatusCode,
				tglr.Params,
				text,
			)
		} else if !tglr.Valid && res.StatusCode != http.StatusBadRequest {
			t.Errorf(
				"expected Bad Request, got %d (test request %+v)",
				res.StatusCode,
				tglr.Params,
			)
		}
	}
}

func TestAddLink(t *testing.T) {
	test_link_requests := []struct {
		Payload map[string]string
		Valid   bool
	}{
		{
			Payload: map[string]string{
				"url":     "",
				"cats":    "test",
				"summary": "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com/wholebunchofextracharswholebunchofextracharswholebunchofextracharswholebunchofextracharswholebunchofextracharswholebunchofextracharswholebunchofextracharswholebunchofextracharswholebunchofextrachars",
				"cats":    "test",
				"summary": "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "notreal",
				"cats":    "test",
				"summary": "bob",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com",
				"cats":    "",
				"summary": "",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com",
				"cats":    "01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
				"summary": "",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com",
				"cats":    "0,1,2,3,4,5,6,7,8,9,0,1,2",
				"summary": "",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com",
				"cats":    "testtest",
				"summary": "01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"url":     "google.com",
				"cats":    "watermelon",
				"summary": "test",
			},
			Valid: true,
		},
		{
			Payload: map[string]string{
				"url":     "about.google.com",
				"cats":    "watermelon",
				"summary": "testy",
			},
			Valid: true,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com/search/howsearchworks/?fg=1",
				"cats":    "watermelon",
				"summary": "testiest",
			},
			Valid: true,
		},
		{
			Payload: map[string]string{
				"url":     "https://www.google.com/search/howsearchworks/features/",
				"cats":    "watermelon",
				"summary": "",
			},
			Valid: true,
		},

		// should fail due to duplicate from previous test with url "google.com"
		{
			Payload: map[string]string{
				"url":     "https://www.google.com",
				"cats":    "test",
				"summary": "",
			},
			Valid: false,
		},
	}

	for _, tr := range test_link_requests {
		pl, _ := json.Marshal(tr.Payload)
		r := httptest.NewRequest(
			http.MethodPost,
			"/links",
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
		AddLink(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != 201 {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			}

			t.Fatalf(
				"expected status code 201, got %d (test request %+v)\n%s", res.StatusCode,
				tr.Payload,
				text,
			)
		} else if !tr.Valid && res.StatusCode != 400 {
			t.Fatalf(
				"expected status code 400, got %d (test request %+v)",
				res.StatusCode,
				tr.Payload,
			)
		}
	}
}

func TestDeleteLink(t *testing.T) {
	var test_requests = []struct {
		LinkID             string
		Valid              bool
		ExpectedStatusCode int
	}{
		// jlk did not submit link 0
		{
			LinkID:             "0",
			Valid:              false,
			ExpectedStatusCode: 403,
		},
		// not a real link
		{
			LinkID:             "-1",
			Valid:              false,
			ExpectedStatusCode: 400,
		},
		// jlk _did_ submit link 7
		{
			LinkID:             "7",
			Valid:              true,
			ExpectedStatusCode: 205,
		},
	}

	for _, tr := range test_requests {
		pl, b := map[string]string{
			"link_id": tr.LinkID,
		}, new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(pl)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(
			http.MethodDelete,
			"/links",
			b,
		)
		r.Header.Set("Content-Type", "application/json")

		ctx := context.Background()
		jwt_claims := map[string]interface{}{
			"login_name": test_login_name,
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		DeleteLink(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != tr.ExpectedStatusCode {
			t.Fatalf(
				"expected status code %d, got %d (test request %+v)",
				tr.ExpectedStatusCode,
				res.StatusCode,
				tr,
			)
		} else if !tr.Valid && res.StatusCode != tr.ExpectedStatusCode {
			t.Fatalf(
				"expected status code %d, got %d (test request %+v)",
				tr.ExpectedStatusCode,
				res.StatusCode,
				tr,
			)
		}
	}
}
