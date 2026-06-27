package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type verifyGuardianConsentInput struct {
	Body struct {
		Token string `json:"token" minLength:"16" maxLength:"128" doc:"Opaque token from the guardian magic link"`
	}
}

type verifyGuardianConsentOutput struct {
	Body struct {
		RegistrationID string `json:"registrationId"`
		Status         string `json:"status"`
	}
}

func registerGuardianConsent(api huma.API, s store.Store, renderer *email.Renderer) {
	huma.Register(api, huma.Operation{
		OperationID: "verify-guardian-consent",
		Method:      "POST",
		Path:        "/guardian-consent/verify",
		Summary:     "Verify a parental-consent magic link",
		Description: "Redeems a guardian-consent token issued during an under-13 registration. " +
			"Marks the token used and flips the registration from pending_guardian_consent → received. " +
			"Idempotent in the sense that an already-used token returns 410 Gone with the same error string.",
		Tags:          []string{"registrations"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *verifyGuardianConsentInput) (*verifyGuardianConsentOutput, error) {
		token, err := s.GetMagicToken(ctx, in.Body.Token)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				// Could be: expired-and-purged-by-TTL, never issued, or
				// revoked. We don't distinguish so a probing attacker can't
				// confirm whether a given token ever existed.
				return nil, huma.Error404NotFound("token not found or expired")
			}
			return nil, err
		}
		if token.Kind != models.TokenKindGuardian {
			// Different magic-link kind ended up at this endpoint —
			// surface the same 404 to avoid leaking that the ID exists.
			return nil, huma.Error404NotFound("token not found or expired")
		}
		now := time.Now().UTC()
		if token.UsedAt != nil {
			return nil, huma.Error410Gone("token already used")
		}
		if !token.ExpiresAt.IsZero() && now.After(token.ExpiresAt) {
			return nil, huma.Error410Gone("token expired")
		}

		// Order matters: mark the token used FIRST (conditional write, blocks
		// concurrent double-redeems), then flip the registration status.
		if err := s.MarkMagicTokenUsed(ctx, token.ID, now); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				return nil, huma.Error410Gone("token already used")
			}
			return nil, err
		}
		if err := s.UpdateRegistrationStatus(ctx, token.ContextID, models.StatusReceived); err != nil {
			return nil, err
		}

		// The registration is now active — send the confirmation email,
		// best-effort. Failures don't affect the response (the registration
		// state is what matters; an admin can re-send manually). Lookups are
		// done after the status flip so a partial failure here doesn't roll
		// the registration back to pending.
		if reg, err := s.GetRegistrationByID(ctx, token.ContextID); err == nil && reg != nil {
			account, accErr := s.GetAccountByID(ctx, token.AccountID)
			runner, runErr := s.GetRunner(ctx, reg.RunnerID)
			race, raceErr := s.GetRace(ctx, reg.RaceID)
			var event *models.Event
			var eventErr error
			if raceErr == nil && race != nil {
				event, eventErr = s.GetEvent(ctx, race.EventID)
			}
			if accErr == nil && runErr == nil && raceErr == nil && eventErr == nil &&
				account != nil && runner != nil && race != nil && event != nil {
				if err := sendRegistrationConfirmationEmail(ctx, renderer, *account, *runner, *race, *event); err != nil {
					log.Printf("registration confirmation email failed (post-guardian): registrationId=%s err=%v", reg.ID, err)
				}
			} else {
				log.Printf("registration confirmation skipped: lookup failed registrationId=%s", token.ContextID)
			}
		}

		out := &verifyGuardianConsentOutput{}
		out.Body.RegistrationID = token.ContextID
		out.Body.Status = models.StatusReceived
		return out, nil
	})
}
