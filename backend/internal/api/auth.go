package api

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// authStep is the wire shape of auth.ChallengeStep so the frontend can branch
// the UI without parsing the challenge JWT.
type authStep string

type loginInput struct {
	Body struct {
		Email    string `json:"email" format:"email" maxLength:"254" doc:"Account email"`
		Password string `json:"password" minLength:"1" maxLength:"256" doc:"Account password"`
	}
}

type loginOutputBody struct {
	Step       authStep `json:"step" doc:"Next step: 'verify' if TOTP already enrolled, 'enroll' if first login"`
	TOTPSecret string   `json:"totpSecret,omitempty" doc:"Base32 TOTP secret — only present when step='enroll'"`
	OTPAuthURI string   `json:"otpauthUri,omitempty" doc:"otpauth:// URI for QR rendering — only present when step='enroll'"`
}

type loginOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      loginOutputBody
}

type totpInput struct {
	Challenge string `cookie:"the-run-auth-challenge"`
	Body      struct {
		Code string `json:"code" minLength:"6" maxLength:"8" doc:"6-digit TOTP code from authenticator"`
	}
}

type sessionUserDTO struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"isAdmin"`
}

type meOutputBody struct {
	User sessionUserDTO `json:"user"`
}

// sessionOutput emits two Set-Cookie headers: the new session and a cleared
// challenge. Huma renders a []string header field as multiple Set-Cookie lines.
type sessionOutput struct {
	SetCookie []string `header:"Set-Cookie"`
	Body      meOutputBody
}

type logoutOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      struct {
		OK bool `json:"ok"`
	}
}

type meInput struct {
	Session string `cookie:"the-run-session"`
}

type meOutput struct {
	Body meOutputBody
}

// registerAuth wires the four auth endpoints. cfg + s are captured via closure
// so handlers stay stateless and unit-testable.
func registerAuth(api huma.API, s store.Store, cfg auth.Config) {
	huma.Register(api, huma.Operation{
		OperationID:   "auth-login",
		Method:        "POST",
		Path:          "/auth/login",
		Summary:       "Begin admin login (password step)",
		Description:   "Verifies the password and issues a TOTP challenge cookie. Follow with POST /auth/totp.",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *loginInput) (*loginOutput, error) {
		addr, err := mail.ParseAddress(in.Body.Email)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}
		email := models.NormalizeEmail(addr.Address)
		account, err := s.GetAccountByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		if account == nil {
			// Constant-ish-time response so unknown emails can't be enumerated
			// via timing. Skip lockout when the account doesn't exist.
			_ = auth.VerifyPassword("$2a$12$abcdefghijklmnopqrstuuKzaG.PPKqL3DEMpL/u5VVAS9DA8nbVKa", in.Body.Password)
			return nil, huma.Error401Unauthorized("invalid credentials")
		}
		if !account.IsAdmin || account.PasswordHash == "" {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}
		if err := auth.EnsureNotLocked(ctx, s, cfg, account.ID, now); err != nil {
			if errors.Is(err, auth.ErrLockedOut) {
				return nil, huma.Error429TooManyRequests("too many failed attempts; try again later")
			}
			return nil, err
		}
		if err := auth.VerifyPassword(account.PasswordHash, in.Body.Password); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				_ = auth.RecordFailedAttempt(ctx, s, cfg, account.ID, now)
				return nil, huma.Error401Unauthorized("invalid credentials")
			}
			return nil, err
		}
		if err := auth.ClearAttempts(ctx, s, account.ID); err != nil {
			return nil, err
		}

		step := auth.StepVerify
		var totpSecret, otpauthURI string
		if account.MFASecret == "" {
			step = auth.StepEnroll
			totpSecret, otpauthURI, err = auth.GenerateTOTP(account.Email)
			if err != nil {
				return nil, err
			}
		}
		challenge, err := auth.IssueChallenge(cfg, account.ID, step, totpSecret, now)
		if err != nil {
			return nil, err
		}
		out := &loginOutput{
			SetCookie: buildCookie(cfg, auth.ChallengeCookieName, challenge, cfg.ChallengeTTL),
		}
		out.Body.Step = authStep(step)
		if step == auth.StepEnroll {
			out.Body.TOTPSecret = totpSecret
			out.Body.OTPAuthURI = otpauthURI
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "auth-totp",
		Method:        "POST",
		Path:          "/auth/totp",
		Summary:       "Complete admin login (TOTP step)",
		Description:   "Reads the challenge cookie set by /auth/login, verifies the TOTP code, and sets a session cookie.",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *totpInput) (*sessionOutput, error) {
		claims, err := auth.ParseChallenge(cfg, in.Challenge)
		if err != nil {
			return nil, huma.Error401Unauthorized("challenge expired or missing")
		}
		account, err := s.GetAccountByID(ctx, claims.Subject)
		if err != nil {
			return nil, huma.Error401Unauthorized("account not found")
		}

		secret := account.MFASecret
		if claims.Step == auth.StepEnroll {
			secret = claims.EnrollSecret
		}
		if secret == "" || !auth.VerifyTOTP(secret, strings.TrimSpace(in.Body.Code)) {
			return nil, huma.Error401Unauthorized("invalid code")
		}

		if claims.Step == auth.StepEnroll {
			account.MFASecret = secret
			if err := s.UpdateAccount(ctx, *account); err != nil {
				return nil, err
			}
		}

		now := time.Now().UTC()
		session, err := auth.IssueSession(cfg, account.ID, account.IsAdmin, now)
		if err != nil {
			return nil, err
		}
		// Touch-last-login is best-effort; don't fail the response if it errors.
		_ = s.TouchAccountLastLogin(ctx, account.ID, now)

		out := &sessionOutput{
			SetCookie: []string{
				buildCookie(cfg, auth.SessionCookieName, session, cfg.SessionTTL),
				buildCookie(cfg, auth.ChallengeCookieName, "", -1),
			},
		}
		out.Body.User = sessionUserDTO{ID: account.ID, Email: account.Email, IsAdmin: account.IsAdmin}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "auth-logout",
		Method:        "POST",
		Path:          "/auth/logout",
		Summary:       "Log out",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, _ *struct{}) (*logoutOutput, error) {
		out := &logoutOutput{
			SetCookie: buildCookie(cfg, auth.SessionCookieName, "", -1),
		}
		out.Body.OK = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "auth-me",
		Method:        "GET",
		Path:          "/auth/me",
		Summary:       "Get the current session",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *meInput) (*meOutput, error) {
		claims, err := auth.ParseSession(cfg, in.Session)
		if err != nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		account, err := s.GetAccountByID(ctx, claims.Subject)
		if err != nil {
			return nil, huma.Error401Unauthorized("account not found")
		}
		out := &meOutput{}
		out.Body.User = sessionUserDTO{ID: account.ID, Email: account.Email, IsAdmin: account.IsAdmin}
		return out, nil
	})
}

// buildCookie renders one Set-Cookie header string. Keeps SameSite/Secure
// flags consistent. maxAge<0 → expires immediately.
func buildCookie(cfg auth.Config, name, value string, maxAge int) string {
	var b strings.Builder
	b.WriteString(name)
	b.WriteString("=")
	b.WriteString(value)
	b.WriteString("; Path=/")
	if cfg.CookieDomain != "" {
		b.WriteString("; Domain=")
		b.WriteString(cfg.CookieDomain)
	}
	if maxAge < 0 {
		b.WriteString("; Max-Age=0")
	} else {
		b.WriteString("; Max-Age=")
		b.WriteString(strconv.Itoa(maxAge))
	}
	b.WriteString("; HttpOnly")
	if cfg.SecureCookies {
		b.WriteString("; Secure")
	}
	b.WriteString("; SameSite=Strict")
	return b.String()
}
