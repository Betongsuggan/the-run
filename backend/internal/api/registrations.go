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
	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// guardianTokenTTL is how long a parental-consent magic link stays valid.
// Chosen so a guardian who registers a kid on Friday evening has the whole
// next weekend to click; matches the "7 days" in the GDPR plan A0.4.
const guardianTokenTTL = 7 * 24 * time.Hour

// underAgeThreshold is the age below which we require an explicit
// guardian-consent magic link (GDPR A0.4). 13 is the Swedish digital-consent
// age; runners aged 13–17 still need a guardian per Swedish minor law but
// don't require the double-opt-in click.
const underAgeThreshold = 13

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
		// Set by the form when the runner is under 13 at the race date —
		// indicates that the submitter is a parent/guardian and consents to
		// the registration. Required (server-side) for the under-13 flow;
		// ignored for everyone else.
		GuardianConsent bool `json:"guardianConsent,omitempty" doc:"Parent/guardian consent ticked (required for under-13 registrations)"`
		// Honeypot field: legitimate clients leave it empty. First line of
		// bot defence; API Gateway throttling on POST /registrations is the
		// second. We stay AWS-only — no third-party challenge widgets.
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

func registerRegistrations(api huma.API, s store.Store, authCfg auth.Config, sender email.Sender) {
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

		// Race + event lookup serves two purposes here: validates that raceId
		// is a real race (was a long-standing TODO) and gives us the event
		// date so we can compute age-at-race for the guardian-consent gate.
		race, err := s.GetRace(ctx, in.Body.RaceID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, huma.Error422UnprocessableEntity("race not found")
			}
			return nil, fmt.Errorf("lookup race: %w", err)
		}
		event, err := s.GetEvent(ctx, race.EventID)
		if err != nil {
			return nil, fmt.Errorf("lookup event: %w", err)
		}
		raceDate, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			return nil, fmt.Errorf("parse event date: %w", err)
		}
		ageAtRace := raceDate.Year() - dob.Year()
		if raceDate.YearDay() < dob.YearDay() {
			ageAtRace--
		}
		requiresGuardianConsent := ageAtRace < underAgeThreshold
		if requiresGuardianConsent && !in.Body.GuardianConsent {
			return nil, huma.Error422UnprocessableEntity("guardian consent is required for runners under " + fmt.Sprintf("%d", underAgeThreshold))
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

		regStatus := models.StatusReceived
		if requiresGuardianConsent {
			regStatus = models.StatusPendingGuardianConsent
		}
		reg := models.Registration{
			ID:        uuid.NewString(),
			RaceID:    in.Body.RaceID,
			RunnerID:  runner.ID,
			Status:    regStatus,
			CreatedAt: now,
		}
		if err := s.CreateRegistration(ctx, reg); err != nil {
			if errors.Is(err, store.ErrAlreadyRegistered) {
				return nil, huma.Error409Conflict("already registered for this race")
			}
			return nil, fmt.Errorf("create registration: %w", err)
		}

		// IDs only — never log name, email, or DOB (GDPR A0.5).
		log.Printf("registration stored: id=%s runnerId=%s raceId=%s accountId=%s status=%s", reg.ID, runner.ID, reg.RaceID, account.ID, reg.Status)

		// Under-13 path: mint a guardian-consent token and email the magic
		// link. We do this AFTER the registration is committed so the
		// token's registrationId points at something real. If the email
		// send fails, the registration still exists in pending state —
		// admin can re-send via a future tool, or the guardian can
		// re-register and we'll dedupe.
		if requiresGuardianConsent {
			tokenID, err := newConsentTokenID()
			if err != nil {
				return nil, fmt.Errorf("generate guardian token: %w", err)
			}
			token := models.GuardianToken{
				ID:                tokenID,
				RegistrationID:    reg.ID,
				GuardianAccountID: account.ID,
				ExpiresAt:         now.Add(guardianTokenTTL),
				CreatedAt:         now,
			}
			if err := s.CreateGuardianToken(ctx, token); err != nil {
				return nil, fmt.Errorf("store guardian token: %w", err)
			}
			if err := sendGuardianConsentEmail(ctx, sender, normalizedEmail, runner.Name, tokenID); err != nil {
				// Don't fail the request — the registration is real and the
				// admin can re-send. Log so we notice in CloudWatch.
				log.Printf("guardian consent email failed: registrationId=%s tokenId=%s err=%v", reg.ID, tokenID, err)
			}
		}

		out := &RegisterOutput{}
		out.Body.ID = reg.ID
		out.Body.RunnerID = runner.ID
		out.Body.Status = reg.Status
		return out, nil
	})
}

// newConsentTokenID returns a 32-byte (256-bit) random opaque string,
// base64-url-encoded without padding. Plenty of entropy to defeat guessing
// even if an attacker could submit millions of attempts.
func newConsentTokenID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// sendGuardianConsentEmail composes the magic-link message and hands it to
// the configured Sender. Body is Swedish-only for now (matches the brand
// language); add an en variant alongside Locale-aware account data later.
func sendGuardianConsentEmail(ctx context.Context, sender email.Sender, toEmail, runnerName, tokenID string) error {
	base := os.Getenv("SITE_BASE_URL")
	if base == "" {
		base = "https://running.rydback.net"
	}
	link := base + "/guardian-consent?token=" + tokenID
	body := "Hej!\n\n" +
		"Du har registrerat " + runnerName + " till ett lopp på Ingmarsöloppet. Eftersom hen är under 13 år behöver vi en målsmans samtycke innan anmälan blir aktiv.\n\n" +
		"Klicka på länken nedan för att bekräfta inom 7 dagar — annars förfaller anmälan automatiskt:\n\n" +
		link + "\n\n" +
		"Om du inte gjort den här anmälan kan du bortse från det här mejlet; ingenting registreras förrän du klickat på länken.\n"
	return sender.Send(ctx, email.Message{
		To:         toEmail,
		Subject:    "Bekräfta målsmans samtycke — Ingmarsöloppet",
		TextBody:   body,
		MessageTag: "guardian-consent",
	})
}
