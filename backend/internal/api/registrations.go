package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

type RegisterInput struct {
	Body struct {
		Name        string `json:"name" minLength:"1" maxLength:"120" doc:"Full name of the registrant"`
		Email       string `json:"email" format:"email" maxLength:"254" doc:"Contact email — used for race comms and to manage your data"`
		DateOfBirth string `json:"dateOfBirth" format:"date" doc:"Birth date (YYYY-MM-DD)"`
		Gender      string `json:"gender" enum:"M,F,X" doc:"Gender code"`
		RaceID      string `json:"raceId" minLength:"1" doc:"ID of the race to register for"`
		// Consent decisions stamped onto the runner / account at create-time.
		// PublicResults is opt-out (Swedish race convention); Marketing is
		// opt-in (Art. 6(1)(a)). Reuse-path semantics: marketing is only
		// captured when a new account is created; publicResults only when a
		// new runner is created. Existing records are never silently
		// overwritten — that's an A1.1 (/my-data) responsibility.
		PublicResults bool `json:"publicResults" doc:"Allow this runner's name to appear in public results (opt-out)"`
		Marketing     bool `json:"marketing" doc:"Receive race news and invitations (opt-in)"`
		// Honeypot field: legitimate clients leave it empty. Real bot
		// protection (Turnstile / hCaptcha) is a TODO — see PROJECT_PLAN.md.
		Website string `json:"website,omitempty" doc:"Leave blank — honeypot"`
	}
}

type RegisterOutput struct {
	Body struct {
		ID       string `json:"id" doc:"Server-assigned registration ID"`
		RunnerID string `json:"runnerId" doc:"ID of the runner (existing or newly created)"`
		Status   string `json:"status" doc:"Lifecycle status, e.g. 'received'"`
	}
}

func registerRegistrations(api huma.API, s store.Store, authCfg auth.Config) {
	huma.Register(api, huma.Operation{
		OperationID:   "register-for-race",
		Method:        "POST",
		Path:          "/registrations",
		Summary:       "Register for a race",
		Description:   "Public race registration. No authentication required; the race must belong to an event in the future.",
		Tags:          []string{"registrations"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *RegisterInput) (*RegisterOutput, error) {
		if strings.TrimSpace(in.Body.Website) != "" {
			// Honeypot tripped — pretend success so the bot doesn't probe further.
			out := &RegisterOutput{}
			out.Body.ID = "ignored"
			out.Body.Status = "received"
			return out, nil
		}

		addr, err := mail.ParseAddress(in.Body.Email)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("email is not a valid address")
		}
		normalizedEmail := models.NormalizeEmail(addr.Address)

		dob, err := time.Parse("2006-01-02", in.Body.DateOfBirth)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("dateOfBirth must be YYYY-MM-DD")
		}
		if dob.After(time.Now()) {
			return nil, huma.Error422UnprocessableEntity("dateOfBirth cannot be in the future")
		}

		now := time.Now().UTC()

		// Find or create the Account this registration belongs to. Non-admin,
		// no password — the runner won't log in yet; the account is just the
		// contact identity for transactional comms and future DSR access.
		account, err := s.GetAccountByEmail(ctx, normalizedEmail)
		if err != nil {
			return nil, fmt.Errorf("lookup account: %w", err)
		}
		if account == nil {
			newAccount := models.Account{
				ID:        uuid.NewString(),
				Email:     normalizedEmail,
				IsAdmin:   false,
				CreatedAt: now,
				Consents: models.AccountConsents{
					Marketing: models.Consent{
						Granted:       in.Body.Marketing,
						At:            now,
						PolicyVersion: authCfg.PrivacyVersion,
					},
				},
			}
			if err := s.CreateAccount(ctx, newAccount); err != nil {
				if errors.Is(err, store.ErrAlreadyExists) {
					// Lost the create race; refetch. The marketing consent we
					// just composed is discarded — the winning request's
					// decision stands.
					account, err = s.GetAccountByEmail(ctx, normalizedEmail)
					if err != nil || account == nil {
						return nil, fmt.Errorf("resolve account after race: %w", err)
					}
				} else {
					return nil, fmt.Errorf("create account: %w", err)
				}
			} else {
				account = &newAccount
			}
		}

		// Dedup within this account: query the byNameDOB GSI (which may return
		// runners from other accounts) and reuse only if AccountID matches.
		nameDobKey := models.NameDobKey(in.Body.Name, in.Body.DateOfBirth)
		candidates, err := s.RunnerByNameDOB(ctx, nameDobKey)
		if err != nil {
			return nil, fmt.Errorf("lookup runner: %w", err)
		}
		var runner *models.Runner
		for i := range candidates {
			if candidates[i].AccountID == account.ID {
				runner = &candidates[i]
				break
			}
		}
		if runner == nil {
			runner = &models.Runner{
				ID:        uuid.NewString(),
				AccountID: account.ID,
				Name:      strings.TrimSpace(in.Body.Name),
				BirthDate: in.Body.DateOfBirth,
				Gender:    in.Body.Gender,
				CreatedAt: now,
				Consents: models.RunnerConsents{
					PublicResults: models.Consent{
						Granted:       in.Body.PublicResults,
						At:            now,
						PolicyVersion: authCfg.PrivacyVersion,
					},
				},
			}
			if err := s.CreateRunner(ctx, *runner); err != nil {
				return nil, fmt.Errorf("create runner: %w", err)
			}
		}

		reg := models.Registration{
			ID:        uuid.NewString(),
			RaceID:    in.Body.RaceID,
			RunnerID:  runner.ID,
			Status:    "received",
			CreatedAt: now,
		}
		if err := s.CreateRegistration(ctx, reg); err != nil {
			if errors.Is(err, store.ErrAlreadyRegistered) {
				return nil, huma.Error409Conflict("already registered for this race")
			}
			return nil, fmt.Errorf("create registration: %w", err)
		}

		// IDs only — never log name, email, or DOB (GDPR A0.5).
		log.Printf("registration stored: id=%s runnerId=%s raceId=%s accountId=%s", reg.ID, runner.ID, reg.RaceID, account.ID)

		out := &RegisterOutput{}
		out.Body.ID = reg.ID
		out.Body.RunnerID = runner.ID
		out.Body.Status = reg.Status
		return out, nil
	})
}
