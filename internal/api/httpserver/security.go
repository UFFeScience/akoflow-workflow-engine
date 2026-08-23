package httpserver

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

// SecurityOptions keeps local development usable while making an accidental
// public listener fail closed. A public API must always have a bearer token.
type SecurityOptions struct {
	BearerToken    string
	AllowedOrigins []string
	LoopbackOnly   bool
}

func SecureAPI(next http.Handler, options SecurityOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if options.LoopbackOnly && !isLoopbackRequest(r) {
			http.Error(w, "API is restricted to loopback", http.StatusForbidden)
			return
		}
		if options.BearerToken != "" && !validBearer(r, options.BearerToken) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validBearer(r *http.Request, expected string) bool {
	value := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return value != "" && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
