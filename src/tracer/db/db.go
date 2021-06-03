package db

import (
	"database/sql"
	"fmt"
	"github.com/maxisme/notifi-backend/tracer"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/trace"
	"net/http"
	"strings"
)

func Exec(r *http.Request, db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	span := getDBSpan(r, "db Exec", query, args...)
	defer span.End()
	res, err := db.Exec(query, args...)
	if err != nil {
		span.RecordError(r.Context(), err)
	}
	return res, err
}

func Query(r *http.Request, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	span := getDBSpan(r, "db Query", query, args...)
	defer span.End()
	return db.Query(query, args...)
}

func QueryRow(r *http.Request, db *sql.DB, query string, args ...interface{}) *sql.Row {
	span := getDBSpan(r, "db QueryRow", query, args...)
	defer span.End()
	return db.QueryRow(query, args...)
}

func getDBSpan(r *http.Request, spanName, query string, args ...interface{}) trace.Span {
	span := tracer.GetSpan(r, spanName)
	span.SetAttributes(
		kv.Key("query").String(strings.TrimSpace(query)),
		kv.Key("args").String(fmt.Sprintf("%v", args)))
	return span
}
