package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Config holds the shared auth-package configuration: the signing secret and
// the cookie/security knobs. Build once at process start; pass by value to
// handlers so unit tests can substitute a deterministic config.
type Config struct {
	JWTSecret     []byte
	CookieDomain  string // empty in dev → host-only cookie
	SecureCookies bool   // false in local dev only
	SessionTTL    int    // seconds; default 8h
	ChallengeTTL  int    // seconds; default 5min
	AttemptTTL    int    // seconds; default 15min
	MaxAttempts   int    // default 5
}

// LoadConfig resolves the JWT signing secret. Order:
//  1. JWT_SECRET env var (LocalStack + unit tests).
//  2. AWS Secrets Manager via JWT_SECRET_ARN (production).
//
// The result is cached for the life of the process; Lambda re-runs LoadConfig
// only on cold starts.
var (
	loadOnce sync.Once
	loadCfg  Config
	loadErr  error
)

func LoadConfig(ctx context.Context) (Config, error) {
	loadOnce.Do(func() {
		secret, err := resolveJWTSecret(ctx)
		if err != nil {
			loadErr = err
			return
		}
		loadCfg = Config{
			JWTSecret:     secret,
			CookieDomain:  os.Getenv("COOKIE_DOMAIN"),
			SecureCookies: os.Getenv("INSECURE_COOKIES") != "1",
			SessionTTL:    8 * 60 * 60,
			ChallengeTTL:  5 * 60,
			AttemptTTL:    15 * 60,
			MaxAttempts:   5,
		}
	})
	return loadCfg, loadErr
}

func resolveJWTSecret(ctx context.Context) ([]byte, error) {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		if len(s) < 32 {
			return nil, errors.New("JWT_SECRET must be at least 32 characters")
		}
		return []byte(s), nil
	}
	arn := os.Getenv("JWT_SECRET_ARN")
	if arn == "" {
		return nil, errors.New("neither JWT_SECRET nor JWT_SECRET_ARN is set")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	sm := secretsmanager.NewFromConfig(cfg)
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	})
	if err != nil {
		return nil, fmt.Errorf("get jwt secret: %w", err)
	}
	if out.SecretString == nil || *out.SecretString == "" {
		return nil, errors.New("jwt secret value is empty")
	}
	if len(*out.SecretString) < 32 {
		return nil, errors.New("jwt secret must be at least 32 characters")
	}
	return []byte(*out.SecretString), nil
}

