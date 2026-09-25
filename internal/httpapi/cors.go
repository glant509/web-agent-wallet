package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const allowedOriginsEnvironment = "AGENT_WALLET_ALLOWED_ORIGINS"

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !isAllowedOrigin(r, origin) {
			http.Error(w, "origin is not allowed", http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Traceparent, X-Trace-ID, X-Span-ID")
		w.Header().Set("Access-Control-Expose-Headers", "Traceparent, X-Trace-ID, X-Span-ID")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(r *http.Request, origin string) bool {
	for _, allowed := range strings.Split(os.Getenv(allowedOriginsEnvironment), ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	parsed, err := url.Parse(origin)
	if err == nil && isSameOriginRequest(r, parsed) {
		return true
	}

	// Packaged clients need CORS while talking to a locally running development
	// server. Cross-origin exceptions do not apply when the API is addressed by a
	// LAN or public hostname; only the same origin or explicitly configured ones do.
	if !isLocalDevelopmentHost(r.Host) {
		return false
	}
	// A wallet page opened directly from the local filesystem has an opaque
	// browser origin serialized as "null". Permit it only while the API itself
	// is addressed through loopback development hosts.
	if origin == "null" {
		return true
	}
	if err != nil {
		return false
	}
	if parsed.Scheme == "chrome-extension" && parsed.Host != "" {
		return true
	}
	if parsed.Scheme == "capacitor" {
		return parsed.Hostname() == "localhost"
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && isLocalDevelopmentHost(parsed.Host)
}

// isSameOriginRequest also understands the forwarding headers normally set by
// Nginx, Caddy and Cloudflare. TLS is commonly terminated by the reverse proxy,
// so r.TLS alone would otherwise make an HTTPS wallet page look cross-origin to
// the Go service and incorrectly reject it with 403.
func isSameOriginRequest(r *http.Request, origin *url.URL) bool {
	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	if forwardedScheme := firstForwardedValue(r.Header.Get("X-Forwarded-Proto")); forwardedScheme != "" {
		requestScheme = strings.ToLower(forwardedScheme)
	}

	requestHost := r.Host
	if forwardedHost := firstForwardedValue(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		requestHost = forwardedHost
	}
	return strings.EqualFold(origin.Scheme, requestScheme) && strings.EqualFold(origin.Host, requestHost)
}

func firstForwardedValue(value string) string {
	value, _, _ = strings.Cut(value, ",")
	return strings.TrimSpace(value)
}

func isLocalDevelopmentHost(value string) bool {
	host := value
	if parsedHost, _, err := net.SplitHostPort(value); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if host == "10.0.2.2" { // Android Emulator alias for the development machine.
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}
