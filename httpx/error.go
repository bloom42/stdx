package httpx

import (
	"fmt"
	"net/http"
)

func ServerInternalErrorPlaintext(w http.ResponseWriter, message *string) {
	var data string

	if message == nil {
		data = "Internal Error"
	} else {
		data = *message
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, data)
}
