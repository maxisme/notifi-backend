package main

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/pkgerrors"
	"net/http"
	"os"
	"time"
)

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func AddLoggingMiddleWare(r *chi.Mux) {
	// chi
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// zerolog
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	r.Use(hlog.NewHandler(logger))
	r.Use(
		hlog.RemoteAddrHandler("ip"), hlog.UserAgentHandler("user_agent"),
		hlog.RefererHandler("referer"), hlog.RequestIDHandler("req_id", "Request-Id"),
	)
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
}

// Handle handles errors and logs them to sentry
func Handle(r *http.Request, err error) {
	if err != nil {
		err = errors.WithStack(errors.Wrap(err, ""))
		logger.Error().Str("req_id", GetRequestID(r)).Stack().Err(err).Msg("")

		// log to sentry
		sentry.CaptureException(err)
		sentry.Flush(time.Second * 5)
	}
}

// WriteError will write a http.Error as well as logging the error
func WriteError(w http.ResponseWriter, r *http.Request, code int, message string) {
	LogError(r, errors.New(message))
	http.Error(w, message, code)
}

func Log(r *http.Request, msg string, level zerolog.Level) {
	logger.WithLevel(level).Str("req_id", GetRequestID(r)).Msg(msg)
}

func LogInfo(r *http.Request, msg string) {
	Log(r, msg, zerolog.InfoLevel)
}

func LogError(r *http.Request, err error) {
	if err != nil {
		Log(r, err.Error(), zerolog.ErrorLevel)

		// log to sentry
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				hub.CaptureMessage(err.Error())
			})
		}
	}
}

func GetRequestID(r *http.Request) string {
	if id, ok := hlog.IDFromRequest(r); ok {
		return fmt.Sprintf("%v", id)
	}
	return ""
}
