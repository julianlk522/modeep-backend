package main

import (
	"log"
	"net/http"
	"os"

	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"

	h "github.com/julianlk522/fitm/handler"
	m "github.com/julianlk522/fitm/middleware"
)

const (
	API_URL = "api.fitm.online:1999"
	// TEST_API_URL = "localhost:1999"
)

var token_auth *jwtauth.JWTAuth

func init() {
	token_auth = jwtauth.New("HS256", []byte(os.Getenv("FITM_JWT_SECRET")), nil)
}

func main() {
	r := chi.NewRouter()
	defer func() {
		if err := http.ListenAndServeTLS(
		API_URL,
			"/etc/letsencrypt/live/api.fitm.online/fullchain.pem",
			"/etc/letsencrypt/live/api.fitm.online/privkey.pem",
			r,
		); err != nil {
			log.Fatal(err)
		}
	}()

	// ROUTER-WIDE MIDDLEWARE
	// LOGGER
	// should go before any other middleware that may change
	// the response, such as middleware.Recoverer
	// (https://github.com/go-chi/chi/blob/6fedde2a70dc2adce0a3dc41b8aebc0b2bec8185/middleware/logger.go#L32C20-L33C46)

	// split logger used to "tee" info from requests with status code 300+
	// to err log file in addition to stdout
	r.Use(m.SplitRequestLogger(m.FileLogFormatter))

	// RATE LIMIT
	// overall
	r.Use(httprate.LimitAll(
		4000,
		time.Minute,
	))
	// by IP
	r.Use(httprate.LimitByIP(
		2400,
		time.Minute,
	))
	r.Use(httprate.LimitByIP(
		100,
		time.Second,
	))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
		},
	}))

	// ROUTES
	// PUBLIC
	r.Post("/signup", h.SignUp)
	r.Post("/login", h.LogIn)
	r.Get("/pic/profile/{file_name}", h.GetProfilePic)
	r.Post("/email-password-reset-link", h.AttemptPasswordReset)
	r.Post("/reset-password", h.ResetPassword)

	r.Get("/pic/preview/{file_name}", h.GetPreviewImg)
	r.Get("/cats", h.GetTopGlobalCats)
	r.Get("/cats/*", h.GetSpellfixMatchesForSnippet)
	r.Get("/contributors", h.GetTopContributors)
	r.Get("/totals", h.GetTotals)

	// CD webhook: application update and refresh
	r.Post("/ghwh", h.HandleGitHubWebhook)

	// OPTIONAL AUTHENTICATION
	// (bearer token used optionally to get IsLiked / IsCopied for links
	// or to authenticate a link click)
	r.Group(func(r chi.Router) {
		r.Use(m.VerifierOptional(token_auth))
		r.Use(m.AuthenticatorOptional(token_auth))
		r.Use(m.JWTContext)

		r.Get("/map/{login_name}", h.GetTreasureMap)
		r.Get("/summaries/{link_id}", h.GetSummaryPage)
		r.Get("/tags/{link_id}", h.GetTagPage)

		r.
			With(m.Pagination).
			Get("/links", h.GetLinks)

		r.
			With(httprate.Limit(
				2,
				time.Second,
				httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
					user_id := r.Context().Value(m.JWTClaimsKey).(map[string]any)["user_id"].(string)
					if user_id != "" {
						return user_id, nil
					}

					return httprate.KeyByIP(r)
				}),
			)).
			Post("/click", h.ClickLink)
	})

	// PROTECTED
	// (bearer token required)
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(token_auth))
		r.Use(jwtauth.Authenticator(token_auth))
		r.Use(m.JWTContext)

		// Users
		r.Put("/about", h.EditAbout)
		r.Post("/pic/profile", h.UploadProfilePic)
		r.Delete("/pic/profile", h.DeleteProfilePic)
		r.Put("/email", h.UpdateEmail)

		// Links
		r.Post("/links", h.AddLink)
		r.Delete("/links", h.DeleteLink)
		r.Post("/links/{link_id}/like", h.LikeLink)
		r.Delete("/links/{link_id}/like", h.UnlikeLink)
		r.Post("/links/{link_id}/copy", h.CopyLink)
		r.Delete("/links/{link_id}/copy", h.UncopyLink)

		// Tags
		r.Post("/tags", h.AddTag)
		r.Put("/tags", h.EditTag)
		r.Delete("/tags", h.DeleteTag)

		// Summaries
		r.Post("/summaries", h.AddSummary)
		r.Delete("/summaries", h.DeleteSummary)
		r.Post("/summaries/{summary_id}/like", h.LikeSummary)
		r.Delete("/summaries/{summary_id}/like", h.UnlikeSummary)
	})
}
