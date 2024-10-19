package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var claims_defaults = map[string]interface{}{
	"user_id": "",
	"login_name": "",
	"iat": nil,
	"exp": nil,
}

// MODIFIED JWT VERIFIER / AUTHENTICATOR
// (requests with no token are allowed,
// but getting link isLiked / isCopied requires a token)
func VerifierOptional(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return VerifyOptional(ja, jwtauth.TokenFromHeader, jwtauth.TokenFromCookie)
}

func VerifyOptional(ja *jwtauth.JWTAuth, findTokenFns ...func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			token, err := VerifyRequestOptional(ja, r, findTokenFns...)
			ctx = jwtauth.NewContext(ctx, token, err)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}

func VerifyRequestOptional(ja *jwtauth.JWTAuth, r *http.Request, findTokenFns ...func(r *http.Request) string) (jwt.Token, error) {
	var tokenString string

	for _, fn := range findTokenFns {
		tokenString = fn(r)
		if tokenString != "" {
			break
		}
	}

	if tokenString == "" {
		return nil, nil
	}

	return jwtauth.VerifyToken(ja, tokenString)
}

func AuthenticatorOptional(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, _, err := jwtauth.FromContext(r.Context())

			// Error decoding token
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return

				// Invalid token
			} else if token != nil && jwt.Validate(token, ja.ValidateOptions()...) != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			// No token or valid token, either way pass through
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hfn)
	}
}

// Retrieve JWT claims if passed in request context or assign empty values
// claims = {"user_id":"1234","login_name":"johndoe", "exp": 1234567890, "iat": 1234567890}
func JWTContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, claims, err := jwtauth.FromContext(r.Context())
		if len(claims) == 0 || err != nil {
			claims = claims_defaults
		} else {
			for k, v := range claims {
				if k == "user_id" || k == "login_name" {
					_, ok := v.(string)
					if !ok {
						claims[k] = claims_defaults[k]
					}
				}
			}
		}

		ctx := context.WithValue(r.Context(), JWTClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
