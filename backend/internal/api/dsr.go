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

	"github.com/BirgerRydback/the-run/backend/internal/audit"
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
	// RecentAudit is the most recent N audit rows for this account,
	// newest-first. Powers the "Your activity" view on /my-data
	// (GDPR A1.1.4). Limit picked client-side; server caps at 200.
	RecentAudit []dsrAuditRowDTO `json:"recentAudit,omitempty"`
}

type dsrAuditRowDTO struct {
	At         string `json:"at"`
	Action     string `json:"action"`
	Actor      string `json:"actor"`
	TargetType string `json:"targetType,omitempty"`
	TargetID   string `json:"targetId,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

type dsrAccountDTO struct {
	ID               string         `json:"id"`
	Email            string         `json:"email"`
	Locale           string         `json:"locale,omitempty"`
	CreatedAt        string         `json:"createdAt"`
	MarketingConsent *dsrConsentDTO `json:"marketingConsent,omitempty"`
	// DeletionPendingUntil is set when the account is in the 30-day grace
	// window after a soft-delete. Dashboard renders a "scheduled for
	// deletion on <date>" banner with a cancel button. Absent for active
	// accounts.
	DeletionPendingUntil string `json:"deletionPendingUntil,omitempty"`
	// InactivityDeletionAt is the date this account would be flagged for
	// auto-deletion if no further activity occurs. Computed as the latest
	// of (createdAt, lastLoginAt, latest registration, latest finished
	// race) + 36 months. Surfaces on /my-data so the user can see when
	// they need to come back. Resets on every login / new registration.
	InactivityDeletionAt string `json:"inactivityDeletionAt,omitempty"`
}

// accountInactivityWindowMonths matches sweepInactiveAccounts in the
// retention Lambda. Bumping one requires bumping the other.
const accountInactivityWindowMonths = 36

type dsrRunnerDTO struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Gender               string         `json:"gender"`
	BirthDate            string         `json:"birthDate"`
	CreatedAt            string         `json:"createdAt"`
	PublicResultsConsent *dsrConsentDTO `json:"publicResultsConsent,omitempty"`
	// Retention tells the user when (or whether) this runner's PII will
	// be auto-anonymized by the retention sweep. Policy is one of:
	//   "kept-indefinitely" — consenting runner, race history preserved
	//   "anonymizes-at"     — non-consenting runner, sliding 36mo window
	//                          from last finished race; AnonymizesAt is set
	//   "no-timer"          — runner has no finished races yet, no timer
	Retention dsrRunnerRetentionDTO `json:"retention"`
}

type dsrRunnerRetentionDTO struct {
	Policy           string `json:"policy"`
	AnonymizesAt     string `json:"anonymizesAt,omitempty"`
	LastFinishedRace string `json:"lastFinishedRace,omitempty"`
}

// nonConsentingRunnerWindowMonths mirrors the retention Lambda's constant.
// Kept in sync by hand — there are only two call sites and changing the
// window is a deliberate policy decision either way.
const nonConsentingRunnerWindowMonths = 36

type dsrConsentDTO struct {
	Granted        bool   `json:"granted"`
	At             string `json:"at"`
	PolicyID       string `json:"policyId,omitempty"`
	PolicyRevision int    `json:"policyRevision,omitempty"`
	PolicyVersion  string `json:"policyVersion"`
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

func registerDSR(api huma.API, s store.Store, authCfg auth.Config, sender email.Sender, recorder audit.Recorder) {
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

		// Note: even accounts in the 30-day grace window get a regular
		// access link here. They're allowed to log in, browse their data,
		// and decide whether to restore — the dashboard surfaces a
		// pending-deletion banner with a one-click "cancel deletion"
		// button (see /dsr/me/cancel-deletion). Restore tokens are still
		// issued at deletion time so an unauthenticated undo path exists
		// via the deletion email, but request-link itself stays neutral.

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
		// Touch lastLoginAt so the inactivity-retention timer (A1.3) treats
		// this as activity. Best-effort: a write failure shouldn't block
		// the session — it just means the timer won't reset this round.
		if err := s.TouchAccountLastLogin(ctx, account.ID, now); err != nil {
			log.Printf("dsr verify: touch lastLoginAt failed: accountId=%s err=%v", account.ID, err)
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  account.ID,
			At:         now,
			Action:     "dsr.session.started",
			Actor:      "user",
			TargetType: "account",
			TargetID:   account.ID,
			Summary:    "Signed in to My data via magic link",
		})
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
		body, err := buildDSRMeBody(ctx, s, recorder)
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
		body, err := buildDSRMeBody(ctx, s, recorder)
		if err != nil {
			return nil, err
		}
		filename := "my-data-" + time.Now().UTC().Format("2006-01-02") + ".json"
		return &dsrExportOutput{
			// Plain "application/json" — Huma matches marshalers by exact
			// content-type string and ships built-in support only for the
			// suffix-free form. Adding "; charset=utf-8" causes a runtime
			// panic ("unknown content type"). UTF-8 is the JSON default
			// per RFC 8259 anyway, so the parameter is redundant.
			ContentType:        "application/json",
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

		// Look up the currently-published policy so each new consent stamp
		// points at the right version. If no policy is published the whole
		// PATCH is rejected — same rationale as the registration handler.
		currentPolicy, err := s.GetPublishedPolicy(ctx)
		if err != nil {
			if errors.Is(err, store.ErrNoPublishedPolicy) {
				return nil, huma.Error503ServiceUnavailable("consent updates are paused — no privacy policy is published")
			}
			return nil, fmt.Errorf("get published policy: %w", err)
		}

		// Marketing — fetch the account, mutate, save. Stamp afresh so the
		// audit trail (consent.At + policyId + revision) is always current.
		if in.Body.Marketing != nil {
			account, err := s.GetAccountByID(ctx, accountID)
			if err != nil {
				return nil, fmt.Errorf("get account: %w", err)
			}
			account.Consents.Marketing = models.Consent{
				Granted:        *in.Body.Marketing,
				At:             now,
				PolicyID:       currentPolicy.ID,
				PolicyRevision: currentPolicy.Revision,
				PolicyVersion:  currentPolicy.Slug,
			}
			if err := s.UpdateAccount(ctx, *account); err != nil {
				return nil, fmt.Errorf("update account: %w", err)
			}
			recorder.Record(ctx, models.AuditRow{
				AccountID:  accountID,
				At:         now,
				Action:     "consent.marketing.update",
				Actor:      "user",
				TargetType: "account",
				TargetID:   accountID,
				Summary:    boolSummary("marketing", *in.Body.Marketing),
			})
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
				Granted:        *update.PublicResults,
				At:             now,
				PolicyID:       currentPolicy.ID,
				PolicyRevision: currentPolicy.Revision,
				PolicyVersion:  currentPolicy.Slug,
			}
			if err := s.UpdateRunner(ctx, *runner); err != nil {
				return nil, fmt.Errorf("update runner: %w", err)
			}
			recorder.Record(ctx, models.AuditRow{
				AccountID:  accountID,
				At:         now,
				Action:     "consent.publicResults.update",
				Actor:      "user",
				TargetType: "runner",
				TargetID:   runner.ID,
				Summary:    boolSummary("publicResults for "+runner.Name, *update.PublicResults),
			})
		}

		body, err := buildDSRMeBody(ctx, s, recorder)
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
		recorder.Record(ctx, models.AuditRow{
			AccountID:  accountID,
			Action:     "runner.update",
			Actor:      "user",
			TargetType: "runner",
			TargetID:   runner.ID,
			Summary:    "Updated runner " + runner.Name,
		})

		body, err := buildDSRMeBody(ctx, s, recorder)
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
		recorder.Record(ctx, models.AuditRow{
			AccountID:  accountID,
			Action:     "locale.update",
			Actor:      "user",
			TargetType: "account",
			TargetID:   accountID,
			Summary:    "Language set to " + in.Body.Locale,
		})
		body, err := buildDSRMeBody(ctx, s, recorder)
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
		recorder.Record(ctx, models.AuditRow{
			AccountID:  account.ID,
			Action:     "email.changed",
			Actor:      "user",
			TargetType: "account",
			TargetID:   account.ID,
			Summary:    "Email changed to " + newEmail,
		})

		out := &dsrConfirmEmailChangeOutput{}
		out.Body.Email = newEmail
		return out, nil
	})

	// ── Erasure ───────────────────────────────────────────────────────────
	// Soft-delete with a 30-day grace window. Sets DeletionPendingUntil on
	// the row(s), cascades non-finished registrations to pending_deletion,
	// mints a restore token, and emails the user an undo link. Hard purge
	// happens later via the retention Lambda.

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-erase-runner",
		Method:        "DELETE",
		Path:          "/dsr/me/runners/{id}",
		Summary:       "Schedule a runner for erasure (30-day undo window)",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
		Middlewares:   dsrMW,
	}, func(ctx context.Context, in *runnerIDInput) (*dsrMeOutput, error) {
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
		now := time.Now().UTC()
		until := now.Add(deletionGraceWindow)
		runner.DeletionPendingUntil = &until
		if err := s.UpdateRunner(ctx, *runner); err != nil {
			return nil, fmt.Errorf("mark runner pending deletion: %w", err)
		}
		// Flip non-finished registrations to pending_deletion so they
		// vanish from admin pipelines too. Finished results stay as-is —
		// they'll get anonymized in place when retention runs.
		if err := cascadePendingDeletion(ctx, s, runner.ID, now); err != nil {
			return nil, err
		}
		if err := issueRestoreLink(ctx, s, sender, accountID, until); err != nil {
			log.Printf("dsr restore link send failed: accountId=%s err=%v", accountID, err)
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  accountID,
			At:         now,
			Action:     "runner.erased",
			Actor:      "user",
			TargetType: "runner",
			TargetID:   runner.ID,
			Summary:    "Scheduled " + runner.Name + " for erasure",
		})
		body, err := buildDSRMeBody(ctx, s, recorder)
		if err != nil {
			return nil, err
		}
		return &dsrMeOutput{Body: *body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-erase-account",
		Method:        "DELETE",
		Path:          "/dsr/me",
		Summary:       "Schedule the entire account + all owned runners for erasure (30-day undo window)",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
		Middlewares:   dsrMW,
	}, func(ctx context.Context, _ *struct{}) (*dsrLogoutOutput, error) {
		accountID := auth.DSRSubject(ctx)
		account, err := s.GetAccountByID(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("get account: %w", err)
		}
		now := time.Now().UTC()
		until := now.Add(deletionGraceWindow)

		// Mark every owned runner first, then the account, so a partial
		// failure leaves runners hidden but the account still reachable
		// for retry. Inverse would orphan runners visible while the
		// account is gone.
		runners, err := s.ListRunnersByAccount(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("list runners: %w", err)
		}
		for i := range runners {
			runners[i].DeletionPendingUntil = &until
			if err := s.UpdateRunner(ctx, runners[i]); err != nil {
				return nil, fmt.Errorf("mark runner pending deletion: %w", err)
			}
			if err := cascadePendingDeletion(ctx, s, runners[i].ID, now); err != nil {
				return nil, err
			}
		}
		account.DeletionPendingUntil = &until
		if err := s.UpdateAccount(ctx, *account); err != nil {
			return nil, fmt.Errorf("mark account pending deletion: %w", err)
		}
		if err := issueRestoreLink(ctx, s, sender, accountID, until); err != nil {
			log.Printf("dsr restore link send failed: accountId=%s err=%v", accountID, err)
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  accountID,
			At:         now,
			Action:     "account.erased",
			Actor:      "user",
			TargetType: "account",
			TargetID:   accountID,
			Summary:    "Scheduled entire account for erasure",
		})

		// End the session so the next /my-data visit goes through
		// request-link → restore-link (or stays gone if they don't act).
		out := &dsrLogoutOutput{
			SetCookie: buildDSRCookie(authCfg, "", -1),
		}
		out.Body.OK = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "dsr-restore",
		Method:        "POST",
		Path:          "/dsr/me/restore",
		Summary:       "Redeem a restore token and clear the pending-deletion flag",
		Description:   "No DSR session required — possession of the restore token (sent to the account's email when deletion was requested) is the proof.",
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
		if token.Kind != models.TokenKindDSRRestore {
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
		// Clear pending-deletion on account.
		if account.DeletionPendingUntil != nil {
			account.DeletionPendingUntil = nil
			if err := s.UpdateAccount(ctx, *account); err != nil {
				return nil, fmt.Errorf("restore account: %w", err)
			}
		}
		// Clear on each owned runner. Unwind cascaded registrations back
		// to a sane status — `received` is the catch-all for "active and
		// pre-race".
		runners, err := s.ListRunnersByAccount(ctx, account.ID)
		if err != nil {
			return nil, fmt.Errorf("list runners: %w", err)
		}
		for _, r := range runners {
			if r.DeletionPendingUntil == nil {
				continue
			}
			r.DeletionPendingUntil = nil
			if err := s.UpdateRunner(ctx, r); err != nil {
				return nil, fmt.Errorf("restore runner: %w", err)
			}
			if err := uncascadePendingDeletion(ctx, s, r.ID); err != nil {
				return nil, err
			}
		}

		recorder.Record(ctx, models.AuditRow{
			AccountID:  account.ID,
			At:         now,
			Action:     "account.restored",
			Actor:      "user",
			TargetType: "account",
			TargetID:   account.ID,
			Summary:    "Cancelled pending erasure",
		})

		// Issue a fresh DSR session so the restored user lands straight on
		// their dashboard — same UX as a normal magic-link verify.
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
		OperationID:   "dsr-cancel-deletion",
		Method:        "POST",
		Path:          "/dsr/me/cancel-deletion",
		Summary:       "Cancel a pending account deletion from within a logged-in session",
		Description:   "Session-authenticated counterpart to /dsr/me/restore. Same effect (clears DeletionPendingUntil on the account + all owned runners, un-cascades pending_deletion registrations) but trusts the DSR session cookie rather than a magic-link token. Used by the dashboard's 'cancel deletion' banner so users who logged in via a fresh magic link don't have to chase the original deletion email.",
		Tags:          []string{"dsr"},
		DefaultStatus: http.StatusOK,
		Middlewares:   dsrMW,
	}, func(ctx context.Context, _ *struct{}) (*dsrMeOutput, error) {
		accountID := auth.DSRSubject(ctx)
		now := time.Now().UTC()

		account, err := s.GetAccountByID(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("get account: %w", err)
		}
		// Idempotent: if the account isn't actually pending deletion (maybe
		// they clicked twice, maybe the magic-link restore already ran),
		// just return the current state.
		if account.DeletionPendingUntil == nil {
			body, err := buildDSRMeBody(ctx, s, recorder)
			if err != nil {
				return nil, err
			}
			return &dsrMeOutput{Body: *body}, nil
		}

		account.DeletionPendingUntil = nil
		if err := s.UpdateAccount(ctx, *account); err != nil {
			return nil, fmt.Errorf("clear account pending deletion: %w", err)
		}
		runners, err := s.ListRunnersByAccount(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("list runners: %w", err)
		}
		for _, r := range runners {
			if r.DeletionPendingUntil == nil {
				continue
			}
			r.DeletionPendingUntil = nil
			if err := s.UpdateRunner(ctx, r); err != nil {
				return nil, fmt.Errorf("restore runner: %w", err)
			}
			if err := uncascadePendingDeletion(ctx, s, r.ID); err != nil {
				return nil, err
			}
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  accountID,
			At:         now,
			Action:     "account.restored",
			Actor:      "user",
			TargetType: "account",
			TargetID:   accountID,
			Summary:    "Cancelled pending erasure from My data",
		})

		body, err := buildDSRMeBody(ctx, s, recorder)
		if err != nil {
			return nil, err
		}
		return &dsrMeOutput{Body: *body}, nil
	})
}

// deletionGraceWindow is how long a soft-deleted account/runner sits before
// the retention Lambda hard-purges PII. Matches the GDPR plan A1.1 spec.
const deletionGraceWindow = 30 * 24 * time.Hour

// cascadePendingDeletion flips a runner's IN-FLIGHT registrations to
// pending_deletion so admins don't see them in pipelines while the runner
// is in their grace window. Terminal statuses (finished / dnf / dns) keep
// their own status — they're meaningful race history. The runner row's
// DeletionPendingUntil flag is what drives anonymization on public
// leaderboards (see expandResult).
func cascadePendingDeletion(ctx context.Context, s store.Store, runnerID string, _ time.Time) error {
	regs, err := s.ListRegistrationsByRunner(ctx, runnerID)
	if err != nil {
		return fmt.Errorf("list registrations: %w", err)
	}
	for _, reg := range regs {
		switch reg.Status {
		case models.StatusFinished, models.StatusDNF, models.StatusDNS, models.StatusPendingDeletion:
			// Terminal status or already cascaded — leave alone.
			continue
		}
		if err := s.UpdateRegistrationStatus(ctx, reg.ID, models.StatusPendingDeletion); err != nil {
			return fmt.Errorf("cascade reg %s: %w", reg.ID, err)
		}
	}
	return nil
}

// uncascadePendingDeletion is the inverse of cascadePendingDeletion. Used by
// the restore flow. Pushes pending_deletion registrations back to the
// catch-all `received` status — anything that was finished was never touched
// so it stays finished.
func uncascadePendingDeletion(ctx context.Context, s store.Store, runnerID string) error {
	regs, err := s.ListRegistrationsByRunner(ctx, runnerID)
	if err != nil {
		return fmt.Errorf("list registrations: %w", err)
	}
	for _, reg := range regs {
		if reg.Status != models.StatusPendingDeletion {
			continue
		}
		if err := s.UpdateRegistrationStatus(ctx, reg.ID, models.StatusReceived); err != nil {
			return fmt.Errorf("uncascade reg %s: %w", reg.ID, err)
		}
	}
	return nil
}

// issueRestoreLink mints a dsr_restore magic token and emails the undo link
// to the account's email (which is still on file during the grace window).
func issueRestoreLink(ctx context.Context, s store.Store, sender email.Sender, accountID string, until time.Time) error {
	account, err := s.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get account for restore link: %w", err)
	}
	tokenID, err := newDSRTokenID()
	if err != nil {
		return fmt.Errorf("generate restore token: %w", err)
	}
	now := time.Now().UTC()
	token := models.MagicToken{
		Kind:      models.TokenKindDSRRestore,
		ID:        tokenID,
		AccountID: accountID,
		ExpiresAt: until, // restore window matches the grace window
		CreatedAt: now,
	}
	if err := s.CreateMagicToken(ctx, token); err != nil {
		return fmt.Errorf("store restore token: %w", err)
	}
	return sendDSRRestoreEmail(ctx, sender, account.Email, tokenID, until)
}

// ── Helpers ───────────────────────────────────────────────────────────────

// buildDSRMeBody collects the account, runners, and registrations for the
// DSR-authenticated subject. Shared between /dsr/me and /dsr/me/export so the
// in-app view and the export file stay byte-identical.
func buildDSRMeBody(ctx context.Context, s store.Store, recorder audit.Recorder) (*dsrMeBody, error) {
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
	if account.DeletionPendingUntil != nil {
		body.Account.DeletionPendingUntil = account.DeletionPendingUntil.UTC().Format(time.RFC3339)
	}
	if c := consentToDTO(account.Consents.Marketing); c != nil {
		body.Account.MarketingConsent = c
	}
	// Compute per-runner "most recent finished race date" once, sharing a
	// race+event cache across runners. Used below to derive each runner's
	// retention status.
	lastFinished := make(map[string]time.Time, len(runners))
	raceCache := map[string]*models.Race{}
	eventCache := map[string]*models.Event{}
	for _, reg := range registrations {
		if reg.Status != models.StatusFinished {
			continue
		}
		race, ok := raceCache[reg.RaceID]
		if !ok {
			r, err := s.GetRace(ctx, reg.RaceID)
			if err != nil || r == nil {
				continue
			}
			race = r
			raceCache[reg.RaceID] = r
		}
		event, ok := eventCache[race.EventID]
		if !ok {
			e, err := s.GetEvent(ctx, race.EventID)
			if err != nil || e == nil {
				continue
			}
			event = e
			eventCache[race.EventID] = e
		}
		date, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			continue
		}
		if existing, ok := lastFinished[reg.RunnerID]; !ok || date.After(existing) {
			lastFinished[reg.RunnerID] = date
		}
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
		dto.Retention = computeRunnerRetention(r, lastFinished[r.ID])
		body.Runners = append(body.Runners, dto)
	}

	// Compute account-level inactivity-deletion date. The sweep counts an
	// account as "active" if any of: it was created within the window,
	// last login within the window, any registration created within the
	// window, any finished race within the window. So the latest of
	// those four points + the window = when it'd be flagged. Mirrors the
	// rules in sweepInactiveAccounts.
	latestActivity := account.CreatedAt
	if account.LastLoginAt != nil && account.LastLoginAt.After(latestActivity) {
		latestActivity = *account.LastLoginAt
	}
	for _, reg := range registrations {
		if reg.CreatedAt.After(latestActivity) {
			latestActivity = reg.CreatedAt
		}
	}
	for _, t := range lastFinished {
		if t.After(latestActivity) {
			latestActivity = t
		}
	}
	body.Account.InactivityDeletionAt = latestActivity.AddDate(0, accountInactivityWindowMonths, 0).UTC().Format(time.RFC3339)
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

	// Recent audit rows — best-effort. If the table read fails we still
	// return the rest of the dashboard; the activity section just renders
	// empty.
	if rows, err := recorder.ListByAccount(ctx, accountID, 50); err == nil {
		body.RecentAudit = make([]dsrAuditRowDTO, 0, len(rows))
		for _, r := range rows {
			body.RecentAudit = append(body.RecentAudit, dsrAuditRowDTO{
				At:         r.At.UTC().Format(time.RFC3339),
				Action:     r.Action,
				Actor:      r.Actor,
				TargetType: r.TargetType,
				TargetID:   r.TargetID,
				Summary:    r.Summary,
			})
		}
	}

	return body, nil
}

// computeRunnerRetention summarizes when (or whether) the retention sweep
// will anonymize this runner's PII. Matches the rules in
// cmd/retention/sweepNonConsentingRunners:
//   - consenting runner → kept indefinitely as race history
//   - non-consenting runner with a finished race → anonymized at
//     lastFinishedRace + nonConsentingRunnerWindowMonths months
//   - runner with no finished races → no timer running yet
//
// The "no-timer" case includes both consenting runners with no races
// (which would be kept indefinitely if they ever race) and non-consenting
// runners with no races (no policy hook to trigger anonymization). The
// frontend distinguishes by also reading publicResultsConsent.granted.
func computeRunnerRetention(r models.Runner, lastFinished time.Time) dsrRunnerRetentionDTO {
	out := dsrRunnerRetentionDTO{}
	if !lastFinished.IsZero() {
		out.LastFinishedRace = lastFinished.UTC().Format("2006-01-02")
	}
	if lastFinished.IsZero() {
		out.Policy = "no-timer"
		return out
	}
	if r.Consents.PublicResults.Granted {
		out.Policy = "kept-indefinitely"
		return out
	}
	out.Policy = "anonymizes-at"
	out.AnonymizesAt = lastFinished.AddDate(0, nonConsentingRunnerWindowMonths, 0).UTC().Format("2006-01-02")
	return out
}

func consentToDTO(c models.Consent) *dsrConsentDTO {
	if c.At.IsZero() && c.PolicyVersion == "" {
		return nil
	}
	return &dsrConsentDTO{
		Granted:        c.Granted,
		At:             c.At.UTC().Format(time.RFC3339),
		PolicyID:       c.PolicyID,
		PolicyRevision: c.PolicyRevision,
		PolicyVersion:  c.PolicyVersion,
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

// boolSummary renders an "X: enabled" / "X: disabled" string for the audit
// row's human-readable Summary field. The frontend translates known action
// codes via i18n, but Summary is the fallback for unknown ones.
func boolSummary(field string, v bool) string {
	if v {
		return field + ": enabled"
	}
	return field + ": disabled"
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

// sendDSRRestoreEmail tells the user their account/runner is queued for
// deletion and gives them the magic link to undo. Sent the moment a
// /dsr/me/runners/{id} or /dsr/me DELETE lands, and also re-issued whenever
// they hit /dsr/request-link while their account is in the grace window.
func sendDSRRestoreEmail(ctx context.Context, sender email.Sender, toEmail, tokenID string, deletionAt time.Time) error {
	base := os.Getenv("SITE_BASE_URL")
	if base == "" {
		base = "https://running.rydback.net"
	}
	link := base + "/my-data/restore?token=" + tokenID
	body := "Hej!\n\n" +
		"Vi har tagit emot din begäran om att radera dina uppgifter hos Ingmarsöloppet.\n\n" +
		"Uppgifterna raderas slutgiltigt den " + deletionAt.Format("2006-01-02") + ". " +
		"Fram till dess är de dolda men kan återställas.\n\n" +
		"Klicka på länken nedan om du ångrar dig och vill behålla dina uppgifter:\n\n" +
		link + "\n\n" +
		"Behöver du ingen ångerlucka? Gör inget — uppgifterna raderas automatiskt på utsatt datum.\n"
	return sender.Send(ctx, email.Message{
		To:         toEmail,
		Subject:    "Dina uppgifter raderas snart — Ingmarsöloppet",
		TextBody:   body,
		MessageTag: "dsr-restore",
	})
}
