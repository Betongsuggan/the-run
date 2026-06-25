// Package turnstile validates Cloudflare Turnstile bot-challenge tokens
// server-side. The public registration endpoint requires a fresh token from
// the browser widget (GDPR A0.7); this package owns the secret-key handling
// and the siteverify round-trip.
//
// When no Turnstile secret is configured at all, Verify becomes a no-op and
// logs a warning once at startup — so LocalStack / unit-test setups don't
// need Cloudflare credentials. Production must set TURNSTILE_SECRET_ARN.
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SiteverifyURL is Cloudflare's token-validation endpoint. Exported so tests
// can point it at a fake server.
var SiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// HTTPClient is the http.Client used to call Cloudflare. Swappable for tests.
var HTTPClient = &http.Client{Timeout: 5 * time.Second}

// Config carries the resolved secret-key + skip flag. Build once at process
// start with LoadConfig; pass by value into handlers.
type Config struct {
	SecretKey        string
	SkipVerification bool
}

var (
	loadOnce sync.Once
	loadCfg  Config
	loadErr  error
)

// LoadConfig resolves the Turnstile secret key. Order:
//  1. TURNSTILE_SECRET env var (LocalStack + unit tests).
//  2. AWS Secrets Manager via TURNSTILE_SECRET_ARN (production).
//  3. Neither set → SkipVerification=true, log a one-time warning so the
//     operator sees that bot protection is OFF.
//
// Cached for the life of the process.
func LoadConfig(ctx context.Context) (Config, error) {
	loadOnce.Do(func() {
		secret, err := resolveSecret(ctx)
		if err != nil {
			loadErr = err
			return
		}
		if secret == "" {
			log.Printf("turnstile: no secret configured (TURNSTILE_SECRET / TURNSTILE_SECRET_ARN); /registrations bot-challenge is DISABLED")
			loadCfg = Config{SkipVerification: true}
			return
		}
		loadCfg = Config{SecretKey: secret}
	})
	return loadCfg, loadErr
}

func resolveSecret(ctx context.Context) (string, error) {
	if s := os.Getenv("TURNSTILE_SECRET"); s != "" {
		return s, nil
	}
	arn := os.Getenv("TURNSTILE_SECRET_ARN")
	if arn == "" {
		return "", nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("load aws config: %w", err)
	}
	sm := secretsmanager.NewFromConfig(cfg)
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	})
	if err != nil {
		return "", fmt.Errorf("get turnstile secret: %w", err)
	}
	if out.SecretString == nil || *out.SecretString == "" {
		// An ARN that resolves to nothing is an operator error, not a green
		// light to skip. Surface it loudly so deploys notice.
		return "", errors.New("turnstile secret value is empty")
	}
	return *out.SecretString, nil
}

// ErrInvalidToken is returned when Cloudflare rejects the token or it's
// missing entirely. Callers map this to HTTP 400.
var ErrInvalidToken = errors.New("turnstile token invalid or missing")

type siteverifyResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
}

// Verify checks the token against Cloudflare's siteverify endpoint. Returns
// ErrInvalidToken for any failure mode short of a transport error (so the
// caller can confidently 400). clientIP is optional but recommended — it
// tightens Cloudflare's risk model.
func Verify(ctx context.Context, cfg Config, token, clientIP string) error {
	if cfg.SkipVerification {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return ErrInvalidToken
	}
	form := url.Values{}
	form.Set("secret", cfg.SecretKey)
	form.Set("response", token)
	if clientIP != "" {
		form.Set("remoteip", clientIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, SiteverifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("siteverify: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode siteverify response: %w", err)
	}
	if !body.Success {
		return ErrInvalidToken
	}
	return nil
}
