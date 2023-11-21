package middleware

import (
	"net/http"

	"github.com/bloom42/stdx/httputils"
)

// var epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

func NoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(httputils.HeaderCacheControl, httputils.CacheControlNoCache)
		// w.Header().Set(httputils.HeaderExpires, epoch) // for Proxies

		next.ServeHTTP(w, r)
	})
}
