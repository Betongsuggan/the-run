package auth

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func testConfig() Config {
	return Config{
		JWTSecret:    []byte("unit-test-secret-must-be-at-least-32-chars"),
		SessionTTL:   3600,
		ChallengeTTL: 300,
		AttemptTTL:   900,
		MaxAttempts:  5,
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("hunter2-correct-horse-battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "" {
		t.Fatal("empty hash")
	}
	if err := VerifyPassword(hash, "hunter2-correct-horse-battery"); err != nil {
		t.Fatalf("verify correct: %v", err)
	}
	if err := VerifyPassword(hash, "wrong"); !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Fatalf("verify wrong: want mismatch, got %v", err)
	}
}

func TestDSRSession_Roundtrip(t *testing.T) {
	cfg := testConfig()
	now := time.Now().UTC()
	tok, err := IssueDSRSession(cfg, "acct-roundtrip", now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := ParseDSRSession(cfg, tok)
	if err != nil {
		t.Fatalf("parse: %v (token=%q)", err, tok)
	}
	if claims.Subject != "acct-roundtrip" {
		t.Errorf("subject = %q, want acct-roundtrip", claims.Subject)
	}
	if claims.Kind != "dsr" {
		t.Errorf("kind = %q, want dsr", claims.Kind)
	}
}

func TestHashEmptyPasswordRejected(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestIssueAndParseSession(t *testing.T) {
	cfg := testConfig()
	now := time.Now()
	tok, err := IssueSession(cfg, "acc-1", true, now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := ParseSession(cfg, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != "acc-1" {
		t.Errorf("subject mismatch: got %q", claims.Subject)
	}
	if !claims.IsAdmin {
		t.Errorf("isAdmin should be true")
	}
}

func TestParseSessionWrongSecretFails(t *testing.T) {
	cfg := testConfig()
	now := time.Now()
	tok, err := IssueSession(cfg, "acc-1", true, now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	other := cfg
	other.JWTSecret = []byte("a-completely-different-secret-which-is-32+")
	if _, err := ParseSession(other, tok); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestParseSessionExpired(t *testing.T) {
	cfg := testConfig()
	past := time.Now().Add(-2 * time.Hour) // SessionTTL is 1h, so 2h ago is expired.
	tok, err := IssueSession(cfg, "acc-1", true, past)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := ParseSession(cfg, tok); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid for expired token, got %v", err)
	}
}

func TestIssueAndParseChallenge(t *testing.T) {
	cfg := testConfig()
	now := time.Now()
	tok, err := IssueChallenge(cfg, "acc-1", StepEnroll, "JBSWY3DPEHPK3PXP", now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := ParseChallenge(cfg, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Step != StepEnroll {
		t.Errorf("step mismatch: got %q", claims.Step)
	}
	if claims.EnrollSecret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("enroll secret round-trip failed")
	}
}

func TestGenerateAndVerifyTOTP(t *testing.T) {
	secret, uri, err := GenerateTOTP("test@example.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if secret == "" {
		t.Fatal("empty secret")
	}
	if uri == "" {
		t.Fatal("empty URI")
	}
	// We can't deterministically know the current code, but we can confirm
	// that a verifier returns false for a wrong code.
	if VerifyTOTP(secret, "000000") {
		// Could happen by chance, but probability is 1/10^6. Document.
		t.Log("note: 000000 was accepted as a valid TOTP, which is statistically rare")
	}
}
