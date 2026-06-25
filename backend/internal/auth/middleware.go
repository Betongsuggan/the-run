package auth

import (
	"context"
	"log"
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

// MaybeAdmin is RequireAdmin's softer sibling: it tries to parse a session
// cookie and attaches the claims to the request context if valid, but never
// rejects the request. Used by public endpoints (e.g. GET /runners) that need
// to project differently for authenticated admins — the public projection
// redacts opt-out runners and minors, the admin projection doesn't.
func MaybeAdmin(cfg Config) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		token := readCookieFromHumaCtx(hctx, SessionCookieName)
		if claims, err := ParseSession(cfg, token); err == nil && claims.IsAdmin {
			hctx = huma.WithContext(hctx, WithClaims(hctx.Context(), claims))
		}
		next(hctx)
	}
}

// dsrCtxKey stashes DSR claims separately from admin claims so the two never
// conflate. An admin using /my-data on their own data has BOTH set; the DSR
// handlers read this key, the admin handlers read ctxKey{}.
type dsrCtxKey struct{}

// WithDSRClaims puts DSR session claims into the context.
func WithDSRClaims(ctx context.Context, claims *DSRSessionClaims) context.Context {
	return context.WithValue(ctx, dsrCtxKey{}, claims)
}

// DSRClaimsFromContext returns the DSR session claims, or nil if the request
// is not authenticated as a DSR session.
func DSRClaimsFromContext(ctx context.Context) *DSRSessionClaims {
	v, _ := ctx.Value(dsrCtxKey{}).(*DSRSessionClaims)
	return v
}

// DSRSubject returns the account id from a DSR session, empty if absent.
func DSRSubject(ctx context.Context) string {
	c := DSRClaimsFromContext(ctx)
	if c == nil {
		return ""
	}
	return c.Subject
}

// RequireDSRSession gates /my-data endpoints. Mirrors RequireAdmin shape:
// reads the DSR cookie, parses+validates, attaches claims, rejects with 401
// on any failure. Distinct cookie name + claim kind from the admin path so
// the two can't be confused even with a bug.
func RequireDSRSession(cfg Config) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		token := readCookieFromHumaCtx(hctx, DSRSessionCookieName)
		claims, err := ParseDSRSession(cfg, token)
		if err != nil {
			// Logged so CloudWatch shows WHY a session was rejected without
			// leaking the reason to the client (which keeps the 401 body
			// stable for unauthenticated callers).
			log.Printf("dsr session rejected: tokenLen=%d err=%v", len(token), err)
			writeAuthError(hctx, http.StatusUnauthorized, "DSR session required")
			return
		}
		hctx = huma.WithContext(hctx, WithDSRClaims(hctx.Context(), claims))
		next(hctx)
	}
}

// readCookieFromHumaCtx parses the Cookie header by name.
//
// Important: the AWS HTTP API v2 → Lambda adapter (aws-lambda-go-api-proxy)
// adds EACH cookie as its own `Cookie:` header line via `Header.Add`, not as
// one comma/semicolon-joined value. Go's `Header.Get("Cookie")` (which is
// what `huma.Context.Header(...)` calls under the hood) returns ONLY the
// first value. So if a request carries multiple cookies (e.g. an admin
// session AND a DSR session), only the cookie in the first header line is
// visible — the others silently vanish. We use EachHeader to walk every
// "Cookie" header line and merge them.
func readCookieFromHumaCtx(hctx huma.Context, name string) string {
	var found string
	hctx.EachHeader(func(hName, hValue string) {
		if found != "" || hName != "Cookie" {
			return
		}
		// Each header line may itself contain multiple cookies separated by
		// "; " (the canonical browser format) — split defensively.
		for _, part := range splitCookies(hValue) {
			if eq := indexByte(part, '='); eq > 0 && part[:eq] == name {
				found = part[eq+1:]
				return
			}
		}
	})
	return found
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
