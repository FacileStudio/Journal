package middleware

import (
	"net"
	"net/http"
	"strings"
)

func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		host, _, err := net.SplitHostPort(request.RemoteAddr)
		if err != nil {
			host = request.RemoteAddr
		}
		remote := net.ParseIP(host)
		if remote != nil && (remote.IsLoopback() || remote.IsPrivate()) {
			if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
				hops := strings.Split(forwarded, ",")
				candidate := strings.TrimSpace(hops[len(hops)-1])
				if ip := net.ParseIP(candidate); ip != nil {
					request.RemoteAddr = ip.String()
				}
			}
		}
		next.ServeHTTP(w, request)
	})
}
