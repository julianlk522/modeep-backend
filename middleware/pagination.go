package middleware

import (
	"context"
	"net/http"
	"strconv"
)

func Pagination(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var page_int int
		var err error

		page := r.URL.Query().Get("page")
		if page == "" {
			page_int = 1
		} else {
			page_int, err = strconv.Atoi(page)
			if err != nil || page_int < 1 {
				page_int = 1
			}
		}
		ctx := context.WithValue(r.Context(), PageKey, page_int)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
