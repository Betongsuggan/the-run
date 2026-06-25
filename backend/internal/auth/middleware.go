package auth

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// ctxKey is the unexported key under which we stash the authenticated session
// claims on the request context. Handlers read it via Subject(ctx).
type ctxKey struct{}

// WithClaims puts session claims into the context. Exported so the login
// handler can re-attach claims to its own response context (rare).
func WithClaims(ctx context.Context, claims *SessionClaims) context.Context {
	return context.WithValue(ctx, ctxKey{}, claims)
}

// ClaimsFromContext returns the authenticated session, or nil if the request
// is unauthenticated.
func ClaimsFromContext(ctx context.Context) *SessionClaims {
	v, _ := ctx.Value(ctxKey{}).(*SessionClaims)
	return v
}

// Subject is a shortcut for "give me the authenticated account id, empty if
// unauthenticated".
func Subject(ctx context.Context) string {
	c := ClaimsFromContext(ctx)
	if c == nil {
		return ""
	}
	return c.Subject
}

// RequireAdmin is the Huma per-operation middleware that gates admin routes.
// It verifies the session cookie, ensures isAdmin=true, and stashes the claims
// in the request context. On any failure: writes 401 directly and aborts the
// downstream handler.
func RequireAdmin(cfg Config) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		// Pull the cookie via the standard library — Huma's huma.Context can
		// give us a net/http.Request via context if needed, but the simplest
		// path is to read the raw cookie header.
		token := readCookieFromHumaCtx(hctx, SessionCookieName)
		claims, err := ParseSession(cfg, token)
		if err != nil {
			writeAuthError(hctx, http.StatusUnauthorized, "not authenticated")
			return
		}
		if !claims.IsAdmin {
			writeAuthError(hctx, http.StatusForbidden, "admin required")
			return
		}
		// Attach claims to the request context so downstream handlers see them.
		hctx = huma.WithContext(hctx, WithClaims(hctx.Context(), claims))
		next(hctx)
	}
}

// readCookieFromHumaCtx parses the Cookie header by name. Huma abstracts away
// http.Request, so we walk the headers directly. (The header is a single line
// with semicolon-separated name=value pairs per RFC 6265.)
func readCookieFromHumaCtx(hctx huma.Context, name string) string {
	header := hctx.Header("Cookie")
	if header == "" {
		return ""
	}
	// http.Request offers Cookies() but we don't have one here; mimic its
	// parsing inline rather than synthesise a request.
	for _, part := range splitCookies(header) {
		if eq := indexByte(part, '='); eq > 0 {
			if part[:eq] == name {
				return part[eq+1:]
			}
		}
	}
	return ""
}

func splitCookies(header string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(header); i++ {
		if header[i] == ';' {
			out = append(out, trimSpace(header[start:i]))
			start = i + 1
		}
	}
	out = append(out, trimSpace(header[start:]))
	return out
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func writeAuthError(hctx huma.Context, status int, detail string) {
	hctx.SetStatus(status)
	hctx.SetHeader("Content-Type", "application/problem+json")
	// Body shape mirrors Huma's standard error: {"detail": "..."}.
	body := `{"status":` + intToString(status) + `,"detail":"` + detail + `"}`
	_, _ = hctx.BodyWriter().Write([]byte(body))
}

func intToString(n int) string {
	switch n {
	case 401:
		return "401"
	case 403:
		return "403"
	case 429:
		return "429"
	}
	// fallback — shouldn't be reached given the call sites.
	return "500"
}
