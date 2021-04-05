package tracer

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"net/http"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := GetInjectSpan(r, fmt.Sprintf("%s %s", r.Method, r.URL.Path), true)
		next.ServeHTTP(w, r)
		span.End()
	})
}

func GetSpan(r *http.Request, spanName string, opts ...trace.StartOption) (span trace.Span) {
	return GetInjectSpan(r, spanName, false, opts...)
}

// GetSpan creates a span as a child of a propagator if r is not nil otherwise it creates a new span
func GetInjectSpan(r *http.Request, spanName string, inject bool, opts ...trace.StartOption) (span trace.Span) {
	if r != nil {
		ctx := propagation.ExtractHTTP(r.Context(), propagation.New(propagation.WithExtractors(trace.B3{})), r.Header)
		ctx, span = global.Tracer("").Start(ctx, spanName, opts...)
		if inject {
			propagation.InjectHTTP(ctx, propagation.New(propagation.WithInjectors(trace.B3{})), r.Header)
		}
	} else {
		_, span = global.Tracer("").Start(context.Background(), spanName, opts...)
	}
	return
}
