package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	m "github.com/julianlk522/modeep/middleware"
	"github.com/julianlk522/modeep/model"
)

func TestAddTag(t *testing.T) {
	// TODO: fix so that payloads are correctly passed to mock request body
	test_tag_requests := []struct {
		Payload map[string]string
		Valid   bool
	}{
		{
			Payload: map[string]string{
				"link_id": "",
				"cats":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "-1",
				"cats":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "101010101010101010101010101010101010101",
				"cats":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "notanint",
				"cats":    "test",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "1",
				"cats":    "",
			},
			Valid: false,
		},
		{
			Payload: map[string]string{
				"link_id": "1",
				"cats":    "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			},
			Valid: false,
		},
		// too many cats
		{
			Payload: map[string]string{
				"link_id": "1",
				"cats":    "0,1,2,3,4,5,6,7,8,9,0,1,2",
			},
			Valid: false,
		},
		// duplicate cats
		{
			Payload: map[string]string{
				"link_id": "1",
				"cats":    "0,1,2,3,3",
			},
			Valid: false,
		},
		// should fail because test user jlk has already tagged link with ID 1
		{
			Payload: map[string]string{
				"link_id": "1",
				"cats":    "testtest",
			},
			Valid: false,
		},
		// should pass because test user jlk has not tagged link with ID 895bed6e-54f9-46d3-ad3f-bb529ba4a1f8
		{
			Payload: map[string]string{
				"link_id": "895bed6e-54f9-46d3-ad3f-bb529ba4a1f8",
				"cats":    "test",
			},
			Valid: true,
		},
	}

	for _, tr := range test_tag_requests {
		pl, _ := json.Marshal(tr.Payload)
		r := httptest.NewRequest(
			http.MethodPost,
			"/tags",
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
		AddTag(w, r)
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
			t.Fatalf(
				"expected status code 400, got %d (test request %+v)",
				res.StatusCode,
				tr.Payload,
			)
		}
	}
}

func TestEditTag(t *testing.T) {
	test_tag_requests := []struct {
		Payload            map[string]string
		Valid              bool
		ExpectedStatusCode int
	}{
		{
			Payload: map[string]string{
				"tag_id": "1",
				"cats":   "",
			},
			Valid:              false,
			ExpectedStatusCode: 400,
		},
		{
			Payload: map[string]string{
				"tag_id": "1",
				"cats":   "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			},
			Valid:              false,
			ExpectedStatusCode: 400,
		},
		// too many cats
		{
			Payload: map[string]string{
				"tag_id": "1",
				"cats":   "0,1,2,3,4,5,6,7,8,9,0,1,2",
			},
			Valid:              false,
			ExpectedStatusCode: 400,
		},
		// duplicate cats
		{
			Payload: map[string]string{
				"tag_id": "1",
				"cats":   "0,1,2,3,3",
			},
			Valid:              false,
			ExpectedStatusCode: 400,
		},
		// should fail because test user jlk did not submit tag with ID 10
		{
			Payload: map[string]string{
				"tag_id": "10",
				"cats":   "testtest",
			},
			Valid:              false,
			ExpectedStatusCode: 403,
		},
		// should pass because test user jlk did submit tag with ID 32
		{
			Payload: map[string]string{
				"tag_id": "32",
				"cats":   "hello,kitty",
			},
			Valid:              true,
			ExpectedStatusCode: 200,
		},
	}

	for _, tr := range test_tag_requests {
		pl, _ := json.Marshal(tr.Payload)
		r := httptest.NewRequest(
			http.MethodPut,
			"/tags",
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
		EditTag(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != tr.ExpectedStatusCode {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			} else {
				t.Fatalf(
					"expected status code %d, got %d (test request %+v)\n%s",
					tr.ExpectedStatusCode,
					res.StatusCode,
					tr.Payload,
					text,
				)
			}
		} else if !tr.Valid && res.StatusCode != tr.ExpectedStatusCode {
			t.Fatalf(
				"expected status code %d, got %d",
				tr.ExpectedStatusCode,
				res.StatusCode,
			)
		}
	}
}

func TestDeleteTag(t *testing.T) {
	var test_requests = []struct {
		TagID              string
		Valid              bool
		ExpectedStatusCode int
	}{
		// test user jlk did not submit tag 11
		{
			TagID:              "11",
			Valid:              false,
			ExpectedStatusCode: 403,
		},
		// not a real tag
		{
			TagID:              "-1",
			Valid:              false,
			ExpectedStatusCode: 400,
		},
		// test user jlk did submit tag 34
		{
			TagID:              "127",
			Valid:              true,
			ExpectedStatusCode: 204,
		},
		// tag with ID 156 is only tag for link 108: should fail
		{
			TagID:              "156",
			Valid:              false,
			ExpectedStatusCode: 400,
		},
	}

	for _, tr := range test_requests {
		pl, b := map[string]string{
			"tag_id": tr.TagID,
		}, new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(pl)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(
			http.MethodDelete,
			"/tags",
			b,
		)
		r.Header.Set("Content-Type", "application/json")

		ctx := context.Background()
		jwt_claims := map[string]any{
			"login_name": TEST_LOGIN_NAME,
		}
		ctx = context.WithValue(ctx, m.JWTClaimsKey, jwt_claims)
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		DeleteTag(w, r)
		res := w.Result()
		defer res.Body.Close()

		if tr.Valid && res.StatusCode != 204 {
			text, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed but unable to read request body bytes")
			}
			t.Fatalf(
				"expected status code 204, got %d (test request %+v) \n%s",
				res.StatusCode,
				tr,
				text,
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

func TestGetSpellfixMatchesForSnippet(t *testing.T) {
	var test_requests = []struct {
		Snippet            string
		OmittedCats        string
		ExpectedStatusCode int
		Results            map[string]int32
	}{
		{
			Snippet:            "test",
			OmittedCats:        "",
			ExpectedStatusCode: 200,
			Results: map[string]int32{
				"test":       21,
				"testing":    2,
				"tech":       2,
				"technology": 1,
			},
		},
		{
			Snippet:            "test",
			OmittedCats:        "test",
			ExpectedStatusCode: 200,
			Results: map[string]int32{
				"testing":    2,
				"tech":       2,
				"technology": 1,
			},
		},
		{
			Snippet:            "test",
			OmittedCats:        "tech,technology",
			ExpectedStatusCode: 200,
			Results: map[string]int32{
				"test":    21,
				"testing": 2,
			},
		},
		{
			Snippet:            "",
			OmittedCats:        "",
			ExpectedStatusCode: 400,
			Results:            nil,
		},
		{
			Snippet:            "",
			OmittedCats:        "test",
			ExpectedStatusCode: 400,
			Results:            nil,
		},
	}

	// define route
	// (otherwise cannot pass URL params without modifying handler implementation)
	r := chi.NewRouter()
	r.Get("/cats/*", GetSpellfixMatchesForSnippet)

	for i, tr := range test_requests {
		req, err := http.NewRequest("GET", "/cats/"+tr.Snippet, nil)
		if err != nil {
			t.Fatal(err)
		}

		if tr.OmittedCats != "" {
			q := req.URL.Query()
			q.Add("omitted", tr.OmittedCats)
			req.URL.RawQuery = q.Encode()
		}

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tr.ExpectedStatusCode {
			b, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatal("failed but unable to read response body bytes")
			}
			t.Fatalf(
				"expected status code %d, got %d (test request %+v) \n%s",
				tr.ExpectedStatusCode,
				w.Code,
				tr,
				string(b),
			)
		}

		// Verify results if valid
		if w.Code > 200 {
			continue
		}

		b, err := io.ReadAll(w.Body)
		if err != nil {
			t.Fatal("failed but unable to read response body bytes")
		}
		var results []model.CatCount
		err = json.Unmarshal(b, &results)
		if err != nil {
			t.Fatal(err)
		}
		for _, res := range results {
			if tr.Results[res.Category] != res.Count {
				t.Fatalf("expected %d for cat %s, got %d (i: %d)", tr.Results[res.Category], res.Category, res.Count, i)
			}
		}
	}
}
