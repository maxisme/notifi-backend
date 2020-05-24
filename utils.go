package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

// Fatal panics errors and sends them to sentry
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

		panic(fmt.Sprintf("Fatal: %s - %s %d", err.Error(), details.Name(), ln)) // TODO check logs as expected
	}
}

// RequiredEnvs verifies envKeys all have values
func RequiredEnvs(envKeys []string) error {
	for _, envKey := range envKeys {
		envValue := os.Getenv(envKey)
		if envValue == "" {
			return fmt.Errorf("missing env variable: '%s'", envKey)
		}
	}
	return nil
}

// UpdateErr returns an error if no rows have been effected
func UpdateErr(res sql.Result, err error) error {
	if err != nil {
		return err
	}

	rowsEffected, err := res.RowsAffected()
	if rowsEffected == 0 {
		return errors.New("no rows effected")
	}
	return err
}
