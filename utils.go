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

func WriteError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	log.Println("http error:" + message)
	_, err := w.Write([]byte(message))
	Handle(err)
}
