package main

import (
	"fmt"
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
	// find where this function has been called from
	pc, _, line, _ := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	calledFrom := fmt.Sprintf("%s line:%d", details.Name(), line)

	log.Printf("HTTP error: message: %s code: %d from:%s \n", message, code, calledFrom)

	// log to sentry
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetExtra("Called From", calledFrom)
			scope.SetExtra("code", code)
			hub.CaptureMessage(message)
		})
	}

	http.Error(w, message, code)
	w.Write([]byte(message))
}
