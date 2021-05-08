package main

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"runtime"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
}

func Log(r *http.Request, level log.Level, args ...interface{}) {
	LogWithSkip(r, level, 3, args...)
}

func getTags(r *http.Request, skip int) log.Fields {
	logTags := log.Fields{}
	pc, _, ln, _ := runtime.Caller(skip)
	details := runtime.FuncForPC(pc)
	if r != nil {
		if len(r.Header.Get("X-B3-Traceid")) > 0 {
			logTags["X-B3-Traceid"] = r.Header.Get("X-B3-Traceid")
		}
	}
	if len(os.Getenv("COMMIT_HASH")) > 0 {
		logTags["commit-hash"] = os.Getenv("COMMIT_HASH")[:7]
	}
	logTags["method"] = details.Name()
	logTags["method-line"] = ln
	return logTags
}

func LogWithSkip(r *http.Request, level log.Level, skip int, args ...interface{}) {
	logTags := getTags(r, skip)

	// write log
	log.WithFields(logTags).Log(level, args...)

	if level < log.InfoLevel {
		// log to sentry
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.Level(level.String()))
				for key := range logTags {
					scope.SetTag(key, fmt.Sprintf("%v", logTags[key]))
				}
				hub.CaptureMessage(fmt.Sprint(args...))
			})
		}
	}
}

// WriteError will write a http.Error as well as logging the error locally and to Sentry
func WriteError(w http.ResponseWriter, r *http.Request, code int, message string) {
	LogWithSkip(r, log.WarnLevel, 3, message)
	http.Error(w, message, code)
	_, _ = w.Write([]byte(message))
}
