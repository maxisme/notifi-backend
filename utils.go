package main

import (
	"github.com/getsentry/sentry-go"
	"log"
	"net/http"
	"runtime"
	"time"
)

func Handle(err error) {
	if err != nil {
		pc, _, _, _ := runtime.Caller(1)
		details := runtime.FuncForPC(pc)
		log.Println("Fatal: " + err.Error() + " - " + details.Name())

		// log to sentry
		sentry.CaptureException(err)
		sentry.Flush(time.Second * 5)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, code int, message string) {
	log.Printf("HTTP error: message: %s code: %d\n", message, code)

	// log to sentry
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetExtra("message", message)
			scope.SetExtra("code", string(code))
			hub.CaptureMessage("Invalid HTTP request")
		})
	}
	w.WriteHeader(code)
	w.Write([]byte(message))
}
