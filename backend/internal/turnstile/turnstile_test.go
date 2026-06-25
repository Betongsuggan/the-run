package turnstile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerify_SkipBypassesEverything(t *testing.T) {
	// Even an empty token must pass when SkipVerification is set — that's the
	// whole point of the flag for local dev / tests.
	if err := Verify(context.Background(), Config{SkipVerification: true}, "", ""); err != nil {
		t.Fatalf("skip mode should never error, got %v", err)
	}
}

func TestVerify_EmptyTokenReturnsInvalidToken(t *testing.T) {
	err := Verify(context.Background(), Config{SecretKey: "test"}, "", "")
	if err != ErrInvalidToken {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}

func TestVerify_CloudflareSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("secret"); got != "shh" {
			t.Errorf("secret = %q, want shh", got)
		}
		if got := r.Form.Get("response"); got != "good-token" {
			t.Errorf("response = %q, want good-token", got)
		}
		if got := r.Form.Get("remoteip"); got != "10.0.0.1" {
			t.Errorf("remoteip = %q, want 10.0.0.1", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	origURL := SiteverifyURL
	SiteverifyURL = srv.URL
	t.Cleanup(func() { SiteverifyURL = origURL })

	if err := Verify(context.Background(), Config{SecretKey: "shh"}, "good-token", "10.0.0.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerify_CloudflareFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":     false,
			"error-codes": []string{"invalid-input-response"},
		})
	}))
	defer srv.Close()

	origURL := SiteverifyURL
	SiteverifyURL = srv.URL
	t.Cleanup(func() { SiteverifyURL = origURL })

	if err := Verify(context.Background(), Config{SecretKey: "shh"}, "bad-token", ""); err != ErrInvalidToken {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}
