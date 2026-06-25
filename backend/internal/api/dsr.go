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

// emailChangeTokenTTL is the window the user has to confirm a new email
// address from their new inbox. 1 day balances "give them time" with
// "limit replay window for an intercepted link".
const emailChangeTokenTTL = 24 * time.Hour

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

// ── PATCH inputs ──────────────────────────────────────────────────────────

type dsrConsentUpdate struct {
	PublicResults *bool `json:"publicResults,omitempty"`
}

type dsrPatchConsentsInput struct {
	Body struct {
		Marketing *bool                       `json:"marketing,omitempty" doc:"New account-level marketing consent (omit to leave unchanged)"`
		PerRunner map[string]dsrConsentUpdate `json:"perRunner,omitempty" doc:"Per-runner consent updates, keyed by runner ID"`
	}
}

type dsrPatchRunnerInput struct {
	ID   string `path:"id" minLength:"1"`
	Body struct {
		Name        *string `json:"name,omitempty" minLength:"1" maxLength:"120"`
		Gender      *string `json:"gender,omitempty" enum:"M,F,X"`
		DateOfBirth *string `json:"dateOfBirth,omitempty" format:"date"`
	}
}

type dsrPatchLocaleInput struct {
	Body struct {
		Locale string `json:"locale" enum:"sv,en" doc:"Preferred language"`
	}
}

type dsrRequestEmailChangeInput struct {
	Body struct {
		NewEmail string `json:"newEmail" format:"email" maxLength:"254"`
	}
}

type dsrConfirmEmailChangeInput struct {
	Body struct {
		Token string `json:"token" minLength:"16" maxLength:"128"`
	}
}

type dsrConfirmEmailChangeOutput struct {
	Body struct {
		Email string `json:"email" doc:"New email now associated with the account"`
	}
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

	huma.Register(api, huma.Operation{
		OperationID: "dsr-patch-consents",
		Method:      "PATCH",
		Path:        "/dsr/me/consents",
		Summary:     "Update marketing + per-runner publicResults consents",
		Tags:        []string{"dsr"},
		Middlewares: dsrMW,
	}, func(ctx context.Context, in *dsrPatchConsentsInput) (*dsrMeOutput, error) {
		accountID := auth.DSRSubject(ctx)
		now := time.Now().UTC()

		// Marketing — fetch the account, mutate, save. Stamp afresh so the
		// audit trail (consent.At + policyVersion) is always current.
		if in.Body.Marketing != nil {
			account, err := s.GetAccountByID(ctx, accountID)
			if err != nil {
				return nil, fmt.Errorf("get account: %w", err)
			}
			account.Consents.Marketing = models.Consent{
				Granted:       *in.Body.Marketing,
				At:            now,
				PolicyVersion: authCfg.PrivacyVersion,
			}
			if err := s.UpdateAccount(ctx, *account); err != nil {
				return nil, fmt.Errorf("update account: %w", err)
			}
		}

		// Per-runner — verify ownership before each write. A naive client
		// that tries to pass another account's runnerId gets 403, not a
		// silent overwrite.
		for runnerID, update := range in.Body.PerRunner {
			if update.PublicResults == nil {
				continue
			}
			runner, err := s.GetRunner(ctx, runnerID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return nil, huma.Error404NotFound("runner not found")
				}
				return nil, fmt.Errorf("get runner: %w", err)
			}
			if runner.AccountID != accountID {
				return nil, huma.Error403Forbidden("runner does not belong to this account")
			}
			runner.Consents.PublicResults = models.Consent{
				Granted:       *update.PublicResults,
				At:            now,
				PolicyVersion: authCfg.PrivacyVersion,
			}
			if err := s.UpdateRunner(ctx, *runner); err != nil {
				return nil, fmt.Errorf("update runner: %w", err)
			}
		}

		body, err := buildDSRMeBody(ctx, s)
		if err != nil {
			return nil, err
		}
		return &dsrMeOutput{Body: *body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "dsr-patch-runner",
		Method:      "PATCH",
		Path:        "/dsr/me/runners/{id}",
		Summary:     "Update a runner's name / gender / date-of-birth",
		Description: "Date-of-birth is locked once this runner has any registration in the finished state — historical results would otherwise re-bucket into a different age category. Returns 409 with detail='date of birth locked' in that case.",
		Tags:        []string{"dsr"},
		Middlewares: dsrMW,
	}, func(ctx context.Context, in *dsrPatchRunnerInput) (*dsrMeOutput, error) {
		accountID := auth.DSRSubject(ctx)
		runner, err := s.GetRunner(ctx, in.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("runner not found")
			}
			return nil, fmt.Errorf("get runner: %w", err)
		}
		if runner.AccountID != accountID {
			return nil, huma.Error403Forbidden("runner does not belong to this account")
		}

		if in.Body.DateOfBirth != nil && *in.Body.DateOfBirth != runner.BirthDate {
			// Locked iff this runner has any finished result. We check the
			// status field across all their registrations.
			regs, err := s.ListRegistrationsByRunner(ctx, runner.ID)
			if err != nil {
				return nil, fmt.Errorf("list registrations: %w", err)
			}
			for _, reg := range regs {
				if reg.Status == models.StatusFinished {
					return nil, huma.Error409Conflict("date of birth locked: this runner has finished results")
				}
			}
			dob, err := time.Parse("2006-01-02", *in.Body.DateOfBirth)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("dateOfBirth must be YYYY-MM-DD")
			}
			if dob.After(time.Now()) {
				return nil, huma.Error422UnprocessableEntity("dateOfBirth cannot be in the future")
			}
			runner.BirthDate = *in.Body.DateOfBirth
		}
		if in.Body.Name != nil {
			runner.Name = strings.TrimSpace(*in.Body.Name)
		}
		if in.Body.Gender != nil {
			runner.Gender = *in.Body.Gender
		}

		if err := s.UpdateRunner(ctx, *runner); err != nil {
			return nil, fmt.Errorf("update runner: %w", err)
		}

		body, err := buildDSRMeBody(ctx, s)
		if err != nil {
			return nil, err
		}
		return &dsrMeOutput{Body: *body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "dsr-patch-locale",
		Method:      "PATCH",
		Path:        "/dsr/me/locale",
		Summary:     "Set the account's preferred language",
		Tags:        []string{"dsr"},
		Middlewares: dsrMW,
	}, func(ctx context.Context, in *dsrPatchLocaleInput) (*dsrMeOutput, error) {
		accountID := auth.DSRSubject(ctx)
		account, err := s.GetAccountByID(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("get account: %w", err)
		}
		account.Locale = in.Body.Locale
		if err := s.UpdateAccount(ctx, *account); err != nil {
			return nil, fmt.Errorf("update account: %w", err)
		}
		body, err := buildDSRMeBody(ctx, s)
		if err != nil {
			return nil, err
		}
		return &dsrMeOutput{Body: *body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-request-email-change",
		Method:        "POST",
		Path:          "/dsr/me/email/request-change",
		Summary:       "Send a confirmation link to a proposed new email",
		Description:   "Always returns 200 — silent if the address is already on another account, to avoid email enumeration via this endpoint. Click the link in the new inbox to actually change.",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
		Middlewares:   dsrMW,
	}, func(ctx context.Context, in *dsrRequestEmailChangeInput) (*struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}, error) {
		out := &struct {
			Body struct {
				OK bool `json:"ok"`
			}
		}{}
		out.Body.OK = true

		addr, err := mail.ParseAddress(in.Body.NewEmail)
		if err != nil {
			return out, nil
		}
		normalizedNew := models.NormalizeEmail(addr.Address)

		accountID := auth.DSRSubject(ctx)
		account, err := s.GetAccountByID(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("get account: %w", err)
		}
		if normalizedNew == account.Email {
			// Same as current — no-op. Silent-200 keeps the response shape
			// uniform so the frontend can show one success state.
			return out, nil
		}

		// If another account already owns the target email, silent-200. We
		// don't want this endpoint to be an oracle for "is this email
		// registered". The confirmation link won't arrive — the user sees
		// nothing — same as the dsr/request-link semantics.
		other, err := s.GetAccountByEmail(ctx, normalizedNew)
		if err != nil {
			return nil, fmt.Errorf("lookup other account: %w", err)
		}
		if other != nil && other.ID != accountID {
			return out, nil
		}

		now := time.Now().UTC()
		tokenID, err := newDSRTokenID()
		if err != nil {
			return nil, fmt.Errorf("generate email-change token: %w", err)
		}
		token := models.MagicToken{
			Kind:      models.TokenKindEmailChange,
			ID:        tokenID,
			AccountID: account.ID,
			ContextID: normalizedNew,
			ExpiresAt: now.Add(emailChangeTokenTTL),
			CreatedAt: now,
		}
		if err := s.CreateMagicToken(ctx, token); err != nil {
			return nil, fmt.Errorf("store email-change token: %w", err)
		}
		if err := sendEmailChangeConfirmation(ctx, sender, normalizedNew, tokenID); err != nil {
			log.Printf("email change confirmation send failed: accountId=%s tokenId=%s err=%v", account.ID, tokenID, err)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-confirm-email-change",
		Method:        "POST",
		Path:          "/dsr/me/email/confirm",
		Summary:       "Confirm a pending email change",
		Description:   "Redeems an email-change token (sent to the proposed new address) and atomically rewrites the EMAIL# sentinel. No DSR session required — possession of the token is the proof of ownership of the new address.",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *dsrConfirmEmailChangeInput) (*dsrConfirmEmailChangeOutput, error) {
		token, err := s.GetMagicToken(ctx, in.Body.Token)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error404NotFound("token not found or expired")
			}
			return nil, err
		}
		if token.Kind != models.TokenKindEmailChange {
			return nil, huma.Error404NotFound("token not found or expired")
		}
		now := time.Now().UTC()
		if token.UsedAt != nil {
			return nil, huma.Error410Gone("token already used")
		}
		if !token.ExpiresAt.IsZero() && now.After(token.ExpiresAt) {
			return nil, huma.Error410Gone("token expired")
		}

		// Mark used FIRST (conditional write) so two simultaneous clicks
		// don't both succeed mid-swap.
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
		oldEmail := account.Email
		newEmail := token.ContextID

		if err := s.ChangeAccountEmail(ctx, account.ID, oldEmail, newEmail); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				// Race: someone else grabbed this email between request-change
				// and confirm. Surface as 409 — the user might retry with a
				// different address.
				return nil, huma.Error409Conflict("email already taken")
			}
			return nil, fmt.Errorf("swap account email: %w", err)
		}

		out := &dsrConfirmEmailChangeOutput{}
		out.Body.Email = newEmail
		return out, nil
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

// sendEmailChangeConfirmation sends the verification link to the NEW address.
// Crucially: the receiver of this email is the proof of ownership. Clicking
// the link is what authorises the swap.
func sendEmailChangeConfirmation(ctx context.Context, sender email.Sender, newEmail, tokenID string) error {
	base := os.Getenv("SITE_BASE_URL")
	if base == "" {
		base = "https://running.rydback.net"
	}
	link := base + "/my-data/confirm-email?token=" + tokenID
	body := "Hej!\n\n" +
		"En begäran har gjorts att byta e-postadress till den här adressen på Ingmarsöloppet. " +
		"Klicka på länken nedan inom 24 timmar för att bekräfta:\n\n" +
		link + "\n\n" +
		"Om du inte gjort den här begäran kan du bortse från mejlet — inget händer förrän du klickat på länken.\n"
	return sender.Send(ctx, email.Message{
		To:         newEmail,
		Subject:    "Bekräfta din nya e-postadress — Ingmarsöloppet",
		TextBody:   body,
		MessageTag: "dsr-email-change",
	})
}
