package httpserver

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"slices"
)

func requestClientAddress(r *http.Request, trusted []netip.Prefix) (string, bool) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", false
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return "", false
	}
	peer = peer.Unmap()
	for _, prefix := range trusted {
		if !prefix.Contains(peer) {
			continue
		}
		values := r.Header.Values("X-Moetopia-Client-IP")
		if len(values) == 0 {
			break
		}
		if len(values) != 1 {
			return "", false
		}
		client, err := netip.ParseAddr(values[0])
		if err != nil {
			return "", false
		}
		return client.Unmap().String(), true
	}
	return peer.String(), true
}

func trustedClientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	trusted = slices.Clone(trusted)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, ok := requestClientAddress(r, trusted)
			if !ok {
				writeJSON(w, 400, map[string]string{"error": "invalid_request"})
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), accountIPKey{}, host)))
		})
	}
}
