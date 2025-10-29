package middleware

import (
	"context"
	"net/http"
	"strconv"
)

func Pagination(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var page uint

		page_params := r.URL.Query().Get("page")
		if page_params == "" {
			page = 1
		} else {
			if page_int, err := strconv.Atoi(page_params); err == nil {
				if page_int > 1 {
					page = uint(page_int)
				}
			}
		}
		ctx := context.WithValue(r.Context(), PageKey, page)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
