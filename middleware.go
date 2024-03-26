package main

import (
	"log"
	"net/http"
	"time"
)

type Middleware func(http.Handler) http.Handler

func CreateMiddlewares(xs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := range xs {
			x := xs[i]
			next = x(next)
		}
		return next
	}
}

type wrappedWrtier struct {
	http.ResponseWriter
	statuscode int
}

func (w *wrappedWrtier) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statuscode = statusCode
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWrtier{
			ResponseWriter: w,
			statuscode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		log.Printf("%d %s %s (%s)", wrapped.statuscode, r.Method, r.URL.Path, time.Since(start))
	})
}
