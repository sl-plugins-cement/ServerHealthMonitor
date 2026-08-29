package web

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Rate Limiter ---

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int           // max requests
	window   time.Duration // per window
}

type visitor struct {
	count     int
	firstSeen time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	// Cleanup old entries periodically
	go func() {
		for range time.Tick(window) {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(v.firstSeen) > rl.window {
		rl.visitors[ip] = &visitor{
			count:     1,
			firstSeen: now,
		}
		return true
	}

	v.count++
	return v.count <= rl.limit
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, v := range rl.visitors {
		if now.Sub(v.firstSeen) > rl.window*2 {
			delete(rl.visitors, ip)
		}
	}
}

// --- Security Headers Middleware ---

func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CSP - Content Security Policy
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Other security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next(w, r)
	}
}

// --- Request Body Size Limiter ---

func maxBodySize(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

// --- CORS Middleware ---

func corsMiddleware(allowedOrigins []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// --- IP helper ---

// getIP returns the client IP for rate limiting and auditing. X-Forwarded-For
// is only trusted when CONSOLE_TRUST_PROXY=1/true — accepting it unconditionally
// lets clients spoof their IP and bypass rate limits.
func getIP(r *http.Request) string {
	if trustProxy() {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			// Take the first IP
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// trustProxy reports whether X-Forwarded-For should be honored (e.g. when the
// console sits behind a reverse proxy that overwrites the header).
func trustProxy() bool {
	v := os.Getenv("CONSOLE_TRUST_PROXY")
	return v == "1" || strings.EqualFold(v, "true")
}

// cookieSecure reports whether session cookies should carry the Secure flag.
// Set CONSOLE_COOKIE_SECURE=1/true when the console is served over HTTPS or
// through a TLS-terminating reverse proxy.
func cookieSecure() bool {
	v := os.Getenv("CONSOLE_COOKIE_SECURE")
	return v == "1" || strings.EqualFold(v, "true")
}

// --- Connectivity check (for the "one-click test" feature) ---

type connResult struct {
	Type    string `json:"type"`
	Target  string `json:"target"`
	Success bool   `json:"success"`
	Latency int64  `json:"latency_ms"`
	Error   string `json:"error,omitempty"`
}

func checkTCP(host string, port int, timeout time.Duration) connResult {
	target := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, timeout)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return connResult{
			Type:    "tcp",
			Target:  target,
			Success: false,
			Latency: latency,
			Error:   err.Error(),
		}
	}
	defer conn.Close()

	return connResult{
		Type:    "tcp",
		Target:  target,
		Success: true,
		Latency: latency,
	}
}

func checkHTTP(url string, timeout time.Duration) connResult {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return connResult{
			Type:    "http",
			Target:  url,
			Success: false,
			Latency: latency,
			Error:   err.Error(),
		}
	}
	defer resp.Body.Close()

	return connResult{
		Type:    "http",
		Target:  url,
		Success: resp.StatusCode >= 200 && resp.StatusCode < 500,
		Latency: latency,
		Error:   fmt.Sprintf("HTTP %d", resp.StatusCode),
	}
}
