package main

import (
	"net/http"
	"os"
)

// ServerKeyMiddleware middleware makes sure the Sec-Key header matches the SERVER_KEY environment variable as
// well as rate limiting requests
func ServerKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Key") != os.Getenv("SERVER_KEY") {
			WriteError(w, r, http.StatusForbidden, "Invalid server key")
			return
		}
		next.ServeHTTP(w, r)
	})
}