package middleware

import (
	"net/http"

	"github.com/bloom42/stdx/httputils"
)

func SetServerHeader(server string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set(httputils.HeaderServer, server)
			next.ServeHTTP(w, req)
		}
		return http.HandlerFunc(fn)
	}
}
