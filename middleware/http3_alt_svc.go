package middleware

import (
	"fmt"
	"net/http"

	"github.com/bloom42/stdx/httputils"
)

func Http3AltSvc(port *string) func(next http.Handler) http.Handler {
	portStr := "443"
	if port != nil {
		portStr = *port
	}
	headerValue := fmt.Sprintf(`h3=":%s"; ma=86400, h3-29=":%s"; ma=86400`, portStr, portStr)

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httputils.HeaderAltSvc, headerValue)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
