// Package api — DSR self-service surface (GDPR A1.1).
//
// The /my-data flow: visitor enters email → POST /dsr/request-link mints a
// magic token, emails the link. Click → /my-data?token=… → POST /dsr/verify
// exchanges the token for a short-lived DSR session cookie. Subsequent
// GET /dsr/me + GET /dsr/me/export run inside that session, gated by
// RequireDSRSession. POST /dsr/logout clears the cookie.
//
// This slice (A1.1.1) ships read-only access + JSON export. Edits, email
// change, erasure, and the audit/activity view land in follow-up slices.

package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// dsrAccessTokenTTL is short by design — the link is one-shot and a fresh one
// is cheap to mint. Limits the window an intercepted link is usable.
const dsrAccessTokenTTL = 30 * time.Minute

// ── Inputs / outputs ──────────────────────────────────────────────────────

type dsrRequestLinkInput struct {
	Body struct {
		Email string `json:"email" format:"email" maxLength:"254" doc:"Account email"`
	}
}

type dsrRequestLinkOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

type dsrVerifyInput struct {
	Body struct {
		Token string `json:"token" minLength:"16" maxLength:"128" doc:"Opaque token from the magic link"`
	}
}

type dsrSessionOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      dsrAccountSummary
}

type dsrAccountSummary struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type dsrLogoutOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      struct {
		OK bool `json:"ok"`
	}
}

// dsrMeOutput is the dashboard payload. Read-only in this slice; later slices
// add editable fields without changing the shape.
type dsrMeOutput struct {
	Body dsrMeBody
}

type dsrMeBody struct {
	Account       dsrAccountDTO        `json:"account"`
	Runners       []dsrRunnerDTO       `json:"runners"`
	Registrations []dsrRegistrationDTO `json:"registrations"`
}

type dsrAccountDTO struct {
	ID               string         `json:"id"`
	Email            string         `json:"email"`
	Locale           string         `json:"locale,omitempty"`
	CreatedAt        string         `json:"createdAt"`
	MarketingConsent *dsrConsentDTO `json:"marketingConsent,omitempty"`
}

type dsrRunnerDTO struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Gender               string         `json:"gender"`
	BirthDate            string         `json:"birthDate"`
	CreatedAt            string         `json:"createdAt"`
	PublicResultsConsent *dsrConsentDTO `json:"publicResultsConsent,omitempty"`
}

type dsrConsentDTO struct {
	Granted       bool   `json:"granted"`
	At            string `json:"at"`
	PolicyVersion string `json:"policyVersion"`
}

type dsrRegistrationDTO struct {
	ID            string `json:"id"`
	RaceID        string `json:"raceId"`
	RunnerID      string `json:"runnerId"`
	Status        string `json:"status"`
	Bib           string `json:"bib,omitempty"`
	FinishSeconds *int   `json:"finishSeconds,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

// dsrExportOutput streams a JSON blob with an attachment header so browsers
// download rather than render it. Body is the same shape as dsrMeBody for
// now — keeps the payload structure single-source-of-truth.
type dsrExportOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               dsrMeBody
}

// ── Registration ──────────────────────────────────────────────────────────

func registerDSR(api huma.API, s store.Store, authCfg auth.Config, sender email.Sender) {
	dsrMW := huma.Middlewares{auth.RequireDSRSession(authCfg)}

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-request-link",
		Method:        "POST",
		Path:          "/dsr/request-link",
		Summary:       "Request a /my-data magic link",
		Description:   "Always returns 200 — does not reveal whether the email is registered (prevents enumeration). If the email matches an account, a magic link is sent via SES.",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *dsrRequestLinkInput) (*dsrRequestLinkOutput, error) {
		out := &dsrRequestLinkOutput{}
		out.Body.OK = true

		addr, err := mail.ParseAddress(in.Body.Email)
		if err != nil {
			// Bad address shape: same silent-200 as unknown account.
			return out, nil
		}
		normalized := models.NormalizeEmail(addr.Address)
		account, err := s.GetAccountByEmail(ctx, normalized)
		if err != nil {
			return nil, fmt.Errorf("lookup account: %w", err)
		}
		if account == nil {
			// No-op silently. Bot probing won't learn anything.
			return out, nil
		}

		now := time.Now().UTC()
		tokenID, err := newDSRTokenID()
		if err != nil {
			return nil, fmt.Errorf("generate dsr token: %w", err)
		}
		token := models.MagicToken{
			Kind:      models.TokenKindDSRAccess,
			ID:        tokenID,
			AccountID: account.ID,
			ExpiresAt: now.Add(dsrAccessTokenTTL),
			CreatedAt: now,
		}
		if err := s.CreateMagicToken(ctx, token); err != nil {
			return nil, fmt.Errorf("store dsr token: %w", err)
		}
		if err := sendDSRAccessEmail(ctx, sender, normalized, tokenID); err != nil {
			// IDs only — never the email or the token in CloudWatch.
			log.Printf("dsr access email failed: accountId=%s tokenId=%s err=%v", account.ID, tokenID, err)
			// Best-effort: still return OK so a flaky SES doesn't reveal
			// account existence by virtue of erroring differently.
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-verify",
		Method:        "POST",
		Path:          "/dsr/verify",
		Summary:       "Exchange a magic-link token for a DSR session cookie",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *dsrVerifyInput) (*dsrSessionOutput, error) {
		token, err := s.GetMagicToken(ctx, in.Body.Token)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("token not found or expired")
			}
			return nil, err
		}
		if token.Kind != models.TokenKindDSRAccess {
			// Wrong kind at this endpoint — return same 404 as missing token
			// so an attacker can't probe to learn which kind a given id is.
			return nil, huma.Error404NotFound("token not found or expired")
		}
		now := time.Now().UTC()
		if token.UsedAt != nil {
			return nil, huma.Error410Gone("token already used")
		}
		if !token.ExpiresAt.IsZero() && now.After(token.ExpiresAt) {
			return nil, huma.Error410Gone("token expired")
		}
		if err := s.MarkMagicTokenUsed(ctx, token.ID, now); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				return nil, huma.Error410Gone("token already used")
			}
			return nil, err
		}

		account, err := s.GetAccountByID(ctx, token.AccountID)
		if err != nil {
			return nil, fmt.Errorf("get account: %w", err)
		}
		jwt, err := auth.IssueDSRSession(authCfg, account.ID, now)
		if err != nil {
			return nil, fmt.Errorf("issue dsr session: %w", err)
		}
		out := &dsrSessionOutput{
			SetCookie: buildDSRCookie(authCfg, jwt, int(auth.DSRSessionTTL.Seconds())),
		}
		out.Body.ID = account.ID
		out.Body.Email = account.Email
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-logout",
		Method:        "POST",
		Path:          "/dsr/logout",
		Summary:       "Clear the /my-data session",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
	}, func(_ context.Context, _ *struct{}) (*dsrLogoutOutput, error) {
		out := &dsrLogoutOutput{
			SetCookie: buildDSRCookie(authCfg, "", -1),
		}
		out.Body.OK = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "dsr-me",
		Method:      "GET",
		Path:        "/dsr/me",
		Summary:     "Read the authenticated account's data",
		Tags:        []string{"dsr"},
		Middlewares: dsrMW,
	}, func(ctx context.Context, _ *struct{}) (*dsrMeOutput, error) {
		body, err := buildDSRMeBody(ctx, s)
		if err != nil {
			return nil, err
		}
		return &dsrMeOutput{Body: *body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "dsr-export",
		Method:      "GET",
		Path:        "/dsr/me/export",
		Summary:     "Export the authenticated account's data as JSON",
		Tags:        []string{"dsr"},
		Middlewares: dsrMW,
	}, func(ctx context.Context, _ *struct{}) (*dsrExportOutput, error) {
		body, err := buildDSRMeBody(ctx, s)
		if err != nil {
			return nil, err
		}
		filename := "my-data-" + time.Now().UTC().Format("2006-01-02") + ".json"
		return &dsrExportOutput{
			ContentType:        "application/json; charset=utf-8",
			ContentDisposition: `attachment; filename="` + filename + `"`,
			Body:               *body,
		}, nil
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────

// buildDSRMeBody collects the account, runners, and registrations for the
// DSR-authenticated subject. Shared between /dsr/me and /dsr/me/export so the
// in-app view and the export file stay byte-identical.
func buildDSRMeBody(ctx context.Context, s store.Store) (*dsrMeBody, error) {
	accountID := auth.DSRSubject(ctx)
	if accountID == "" {
		// Defensive — middleware should guarantee this is populated.
		return nil, huma.Error401Unauthorized("DSR session required")
	}
	account, err := s.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	runners, err := s.ListRunnersByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list runners: %w", err)
	}
	// Gather registrations across all runners. For a small race with a few
	// runners per account, the per-runner Query is fine. If this grows,
	// move to a single byAccount registration GSI.
	var registrations []models.Registration
	for _, r := range runners {
		regs, err := s.ListRegistrationsByRunner(ctx, r.ID)
		if err != nil {
			return nil, fmt.Errorf("list registrations: %w", err)
		}
		registrations = append(registrations, regs...)
	}

	body := &dsrMeBody{
		Account: dsrAccountDTO{
			ID:        account.ID,
			Email:     account.Email,
			Locale:    account.Locale,
			CreatedAt: account.CreatedAt.UTC().Format(time.RFC3339),
		},
		Runners:       make([]dsrRunnerDTO, 0, len(runners)),
		Registrations: make([]dsrRegistrationDTO, 0, len(registrations)),
	}
	if c := consentToDTO(account.Consents.Marketing); c != nil {
		body.Account.MarketingConsent = c
	}
	for _, r := range runners {
		dto := dsrRunnerDTO{
			ID:        r.ID,
			Name:      r.Name,
			Gender:    r.Gender,
			BirthDate: r.BirthDate,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		}
		if c := consentToDTO(r.Consents.PublicResults); c != nil {
			dto.PublicResultsConsent = c
		}
		body.Runners = append(body.Runners, dto)
	}
	for _, reg := range registrations {
		body.Registrations = append(body.Registrations, dsrRegistrationDTO{
			ID:            reg.ID,
			RaceID:        reg.RaceID,
			RunnerID:      reg.RunnerID,
			Status:        reg.Status,
			Bib:           reg.Bib,
			FinishSeconds: reg.FinishSeconds,
			CreatedAt:     reg.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return body, nil
}

func consentToDTO(c models.Consent) *dsrConsentDTO {
	if c.At.IsZero() && c.PolicyVersion == "" {
		return nil
	}
	return &dsrConsentDTO{
		Granted:       c.Granted,
		At:            c.At.UTC().Format(time.RFC3339),
		PolicyVersion: c.PolicyVersion,
	}
}

func newDSRTokenID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// buildDSRCookie renders one Set-Cookie line for the DSR session. Mirrors
// buildCookie from auth.go but parameterised on the DSR cookie name.
func buildDSRCookie(cfg auth.Config, value string, maxAge int) string {
	var b strings.Builder
	b.WriteString(auth.DSRSessionCookieName)
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
		b.WriteString(itoa(maxAge))
	}
	b.WriteString("; HttpOnly")
	if cfg.SecureCookies {
		b.WriteString("; Secure")
	}
	b.WriteString("; SameSite=Strict")
	return b.String()
}

// itoa is a tiny shim so we don't import strconv just for one call here.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}

// sendDSRAccessEmail composes the magic-link email and hands it to the
// configured Sender. Swedish body; the user can switch locales once we wire
// account.Locale through the sender (future slice).
func sendDSRAccessEmail(ctx context.Context, sender email.Sender, toEmail, tokenID string) error {
	base := os.Getenv("SITE_BASE_URL")
	if base == "" {
		base = "https://running.rydback.net"
	}
	link := base + "/my-data?token=" + tokenID
	body := "Hej!\n\n" +
		"Du har begärt att hantera dina uppgifter hos Ingmarsöloppet. Klicka på länken nedan inom 30 minuter för att logga in:\n\n" +
		link + "\n\n" +
		"Om du inte begärt det här mejlet kan du bortse från det.\n"
	return sender.Send(ctx, email.Message{
		To:         toEmail,
		Subject:    "Hantera dina uppgifter — Ingmarsöloppet",
		TextBody:   body,
		MessageTag: "dsr-access",
	})
}
