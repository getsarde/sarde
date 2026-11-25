package server

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/getsarde/sarde/internal/devlog"
)

// hostAllowed reports whether a request's Host may reach the dev server. It
// blocks DNS rebinding: a web page on a domain that re-resolves to this
// machine sends its own domain as Host and is refused, so it cannot read the
// preview (drafts included). Accepted: an IP address (LAN access with
// --host 0.0.0.0 goes through the machine's IP), localhost and *.localhost
// names, which no outside page can own, and the host the server was bound
// to. An empty Host (an HTTP/1.0 client) is accepted: browsers always send
// one.
func hostAllowed(hostport, bound string) bool {
	if hostport == "" {
		return true
	}
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
	if host == "" {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	return strings.EqualFold(host, strings.Trim(bound, "[]"))
}

// hostGuard answers 403 to a request whose Host fails hostAllowed, and
// warns once per refused host.
func (ds *DevServer) hostGuard(next http.Handler) http.Handler {
	var warned sync.Map
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostAllowed(r.Host, ds.host) {
			if _, seen := warned.LoadOrStore(strings.ToLower(r.Host), true); !seen {
				devlog.Warn("server", "Refused a request for host %q: the dev server answers only localhost, IP addresses and %s", r.Host, ds.host)
			}
			http.Error(w, "Forbidden: unknown host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
