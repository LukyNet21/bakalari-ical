package main

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("--> %s %s from %s (ua=%q)", r.Method, r.URL.RequestURI(), r.RemoteAddr, r.UserAgent())

		rec := &statusRecorder{ResponseWriter: w, status: 0}
		next.ServeHTTP(rec, r)

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		log.Printf("<-- %s %s %d (%d bytes) in %s",
			r.Method, r.URL.Path, rec.status, rec.bytes, time.Since(start).Round(time.Millisecond))
	})
}
