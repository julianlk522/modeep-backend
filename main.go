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

var (
	token_auth *jwtauth.JWTAuth
	api_url = "api.fitm.online:1999"
	// test_api_url = "localhost:1999"
)

func init() {
	token_auth = jwtauth.New("HS256", []byte(os.Getenv("FITM_JWT_SECRET")), nil)
}

func main() {
	r := chi.NewRouter()
	defer func() {
		if err := http.ListenAndServeTLS(
		api_url,
			"/etc/letsencrypt/live/api.fitm.online/fullchain.pem",
			"/etc/letsencrypt/live/api.fitm.online/privkey.pem",
			r,
		); err != nil {
			log.Fatal(err)
		}
		// if err := http.ListenAndServe(test_api_url, r); err != nil {
		// 	log.Fatal(err)
		// }
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
	// per minute (overall)
	// absolute max before all traffic stopped
	r.Use(httprate.LimitAll(
		4000,
		time.Minute,
	))
	// per minute (IP)
	// needs to cover all concurrent traffic coming from the frontend
	// shared across all users (hence it being really high)
	r.Use(httprate.Limit(
		2400,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
	))
	// per second (IP)
	// (stop short bursts quickly)
	r.Use(httprate.Limit(
		100,
		1*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByIP),
	))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
		},
		// Debug: true,
	}))

	// ROUTES
	// PUBLIC
	r.Post("/signup", h.SignUp)
	r.Post("/login", h.LogIn)
	r.Get("/pic/{file_name}", h.GetProfilePic)

	r.Get("/cats", h.GetTopGlobalCats) // includes subcats
	r.Get("/cats/*", h.GetSpellfixMatchesForSnippet)
	r.Get("/contributors", h.GetTopContributors)

	// CD webhook: application update and refresh
	r.Post("/ghwh", h.HandleGitHubWebhook)

	// OPTIONAL AUTHENTICATION
	// (bearer token used optionally to get IsLiked / IsCopied for links)
	r.Group(func(r chi.Router) {
		r.Use(m.VerifierOptional(token_auth))
		r.Use(m.AuthenticatorOptional(token_auth))
		r.Use(m.JWTContext)

		r.Get("/map/{login_name}", h.GetTreasureMap)

		r.
			With(m.Pagination).
			Get("/links", h.GetLinks)

		r.Get("/summaries/{link_id}", h.GetSummaryPage)
		r.Get("/tags/{link_id}", h.GetTagPage)
	})

	// PROTECTED
	// (bearer token required)
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(token_auth))
		r.Use(jwtauth.Authenticator(token_auth))
		r.Use(m.JWTContext)

		// Users
		r.Put("/about", h.EditAbout)
		r.Post("/pic", h.UploadProfilePic)
		r.Delete("/pic", h.DeleteProfilePic)

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
