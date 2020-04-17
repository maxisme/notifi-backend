package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

// Fatal panics errors and logs them to sentry
func Fatal(err error) {
	if err != nil {
		// log err to sentry
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelFatal)
			sentry.CaptureException(err)
		})
		sentry.Flush(time.Second * 5)

		pc, _, ln, _ := runtime.Caller(1)
		details := runtime.FuncForPC(pc)

		panic(fmt.Sprintf("Fatal: %s - %s %d", err.Error(), details.Name(), ln))
	}
}

// LogErr logs errors and sends them to sentry
func LogErr(err error) {
	if err != nil {
		// log err to sentry
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelError)
			sentry.CaptureException(err)
		})
		sentry.Flush(time.Second * 5)

		pc, _, ln, _ := runtime.Caller(1)
		details := runtime.FuncForPC(pc)

		log.Printf("Error: %s - %s %d", err.Error(), details.Name(), ln)
	}
}

// WriteError will write a http.Error as well as logging the error locally and to Sentry
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
			scope.SetExtra("Header Code", code)
			hub.CaptureMessage(message)
		})
	}

	http.Error(w, message, code)
}

// RequiredEnvs verifies envKeys all have values
func RequiredEnvs(envKeys []string) error {
	for _, envKey := range envKeys {
		envValue := os.Getenv(envKey)
		if envValue == "" {
			return fmt.Errorf("missing env '%s'", envKey)
		}
	}
	return nil
}
