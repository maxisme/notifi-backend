package tracer

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"net"
	"net/http"
	"os"
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

func getTags() []kv.KeyValue {
	tags := []kv.KeyValue{
		kv.String("commit-hash", os.Getenv("COMMIT_HASH")),
	}
	host, _ := os.Hostname()
	tags = append(tags, kv.String("host", host))
	ips, _ := net.LookupIP(host)
	for id, addr := range ips {
		if ipv4 := addr.To4(); ipv4 != nil {
			tags = append(tags, kv.String(fmt.Sprintf("hostname-ip-%d", id), fmt.Sprintf("%v", ipv4)))
		}
	}
	return tags
}
