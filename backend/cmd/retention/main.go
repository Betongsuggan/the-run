// Command retention runs the GDPR A1.1.3 retention sweep. Invoked daily by
// EventBridge; the Lambda walks the accounts and runners tables looking for
// rows whose DeletionPendingUntil has passed, and anonymizes / drops the
// associated PII according to the rules in
// /home/birgerrydback/.claude/plans/nice-work-can-we-cozy-blum.md.
//
// Cascade rules:
//   - Runner past grace window: name="Anonym", drop birthDate/gender, keep
//     the UUID + accountId link so finished result rows remain coherent on
//     leaderboards (the "anonymize in place" outcome from A0.8). Drop the
//     publicResults consent record — there's no PII left to consent about.
//     Pending/DNS/DNF registrations for this runner: hard-deleted. Finished
//     registrations: kept (their names already read as "Anonym" via the
//     anonymized runner).
//   - Account past grace window: drop email row + sentinel, all PII fields,
//     all owned runners (same cascade as above). Any magic tokens for the
//     account become unredeemable (TTL purges them eventually).
//
// The Lambda is idempotent: a row past grace that was already anonymized
// gets skipped because DeletionPendingUntil is nil after the purge.
//
// Logged actions are IDs only (GDPR A0.5).
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/BirgerRydback/the-run/backend/internal/audit"
	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/gdpr"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// Time-based retention windows come from internal/gdpr (GDPR A1.3, schedule
// in B1.4). The ROPA generator reads from the same constants, so the legal
// record can't drift from the sweep's actual behaviour.
const (
	nonConsentingRunnerWindow     = gdpr.NonConsentingRunnerWindowMonths
	nonFinishedRegistrationWindow = gdpr.NonFinishedRegistrationMonths
	auditLogWindow                = gdpr.AuditLogMonths
	accountInactivityWindow       = gdpr.AccountInactivityMonths
)

const inactivityDeletionGrace = gdpr.InactivityDeletionGrace

func main() {
	if isLambdaRuntime() {
		lambda.Start(handler)
		return
	}
	// Local invocation path — handy for testing against LocalStack with
	// `go run ./cmd/retention`.
	if err := handler(context.Background()); err != nil {
		log.Fatalf("retention sweep failed: %v", err)
	}
}

func isLambdaRuntime() bool {
	// AWS_LAMBDA_RUNTIME_API is set by the Lambda execution environment.
	// Same probe used in cmd/api/main.go.
	return os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}

func handler(ctx context.Context) error {
	s, err := store.NewDynamoStoreFromEnv(ctx)
	if err != nil {
		return err
	}
	recorder, _, err := audit.NewRecorderFromEnv(ctx)
	if err != nil {
		return err
	}
	sender, ses, err := email.NewSenderFromEnv(ctx)
	if err != nil {
		return err
	}
	log.Printf("retention: email sender initialised (ses=%t)", ses)
	renderer := email.NewRenderer(s, sender)
	now := time.Now().UTC()

	// DSR-driven soft-delete sweeps — anything past its 30-day grace
	// window gets hard-purged.
	if err := sweepRunners(ctx, s, now); err != nil {
		// Don't abort on first error — log + continue so a single bad row
		// doesn't keep the whole batch from running. Lambda retries on
		// non-nil return; we'd rather surface partial progress + a clear
		// per-row error than nuke the next 24h of sweeps.
		log.Printf("retention: sweepRunners errored: %v", err)
	}
	if err := sweepAccounts(ctx, s, now); err != nil {
		log.Printf("retention: sweepAccounts errored: %v", err)
	}

	// Time-based retention sweeps (A1.3).
	if err := sweepNonConsentingRunners(ctx, s, recorder, now); err != nil {
		log.Printf("retention: sweepNonConsentingRunners errored: %v", err)
	}
	if err := sweepNonFinishedRegistrations(ctx, s, now); err != nil {
		log.Printf("retention: sweepNonFinishedRegistrations errored: %v", err)
	}
	if err := sweepInactiveAccounts(ctx, s, recorder, renderer, now); err != nil {
		log.Printf("retention: sweepInactiveAccounts errored: %v", err)
	}
	if n, err := recorder.PurgeBefore(ctx, now.AddDate(0, -auditLogWindow, 0)); err != nil {
		log.Printf("retention: purge audit errored: %v", err)
	} else if n > 0 {
		log.Printf("retention: purged %d audit rows older than %d months", n, auditLogWindow)
	}
	return nil
}

// sweepNonConsentingRunners anonymizes the PII of opted-out runners whose
// most recent finished race is older than the 36-month sliding window.
// Anonymizing leaves the runner row in place with name="Anonym" so finished
// result rows remain coherent on public leaderboards (already redacted
// via the consent flag, now persisted that way).
//
// A runner who never finished a race doesn't trigger this rule — there's
// no "last activity" to base the window on. Their data sits until either
// they participate (sliding window starts) or they erase via /my-data.
func sweepNonConsentingRunners(ctx context.Context, s store.Store, recorder audit.Recorder, now time.Time) error {
	threshold := now.AddDate(0, -nonConsentingRunnerWindow, 0)
	runners, err := s.ListRunners(ctx)
	if err != nil {
		return err
	}
	for _, r := range runners {
		if r.Consents.PublicResults.Granted {
			continue
		}
		// Skip rows we've already anonymized. Safe heuristic: real names
		// never equal the literal string "Anonym" (we'd reject that
		// server-side if anyone tried to register it).
		if r.Name == "Anonym" || r.Name == "" {
			continue
		}
		// DSR pending-deletion rows are handled by sweepRunners; don't
		// double-purge.
		if r.DeletionPendingUntil != nil {
			continue
		}
		lastFinish, ok := mostRecentFinishedRace(ctx, s, r.ID)
		if !ok {
			continue
		}
		if lastFinish.After(threshold) {
			continue
		}
		if err := purgeRunner(ctx, s, r); err != nil {
			log.Printf("retention: anonymize non-consenting runner id=%s failed: %v", r.ID, err)
			continue
		}
		log.Printf("retention: anonymized non-consenting runner id=%s accountId=%s lastRace=%s",
			r.ID, r.AccountID, lastFinish.Format("2006-01-02"))
		// Audit row on the owning account so the user can see in /my-data
		// that this happened. (The account itself stays — only the runner
		// is anonymized.)
		if r.AccountID != "" {
			recorder.Record(ctx, models.AuditRow{
				AccountID:  r.AccountID,
				At:         now,
				Action:     "retention.runner.anonymized",
				Actor:      "system",
				TargetType: "runner",
				TargetID:   r.ID,
				Summary:    "Anonymized per 36-month retention policy",
			})
		}
	}
	return nil
}

// mostRecentFinishedRace looks up the runner's finished registrations,
// resolves each to its race's event date, and returns the latest. ok=false
// means the runner has no finished races (the time-based rule doesn't
// apply).
func mostRecentFinishedRace(ctx context.Context, s store.Store, runnerID string) (time.Time, bool) {
	regs, err := s.ListRegistrationsByRunner(ctx, runnerID)
	if err != nil {
		return time.Time{}, false
	}
	var latest time.Time
	for _, reg := range regs {
		if reg.Status != models.StatusFinished {
			continue
		}
		race, err := s.GetRace(ctx, reg.RaceID)
		if err != nil || race == nil {
			continue
		}
		event, err := s.GetEvent(ctx, race.EventID)
		if err != nil || event == nil {
			continue
		}
		date, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			continue
		}
		if date.After(latest) {
			latest = date
		}
	}
	if latest.IsZero() {
		return time.Time{}, false
	}
	return latest, true
}

// sweepNonFinishedRegistrations drops DNS / DNF registrations whose race
// is older than the 6-month window. These rows don't carry placement or
// history value; they're just a residual link between a runner and a
// race they didn't actually run.
//
// Finished registrations are NOT touched — they're race history. The
// runner's name on finished rows is anonymized via the consent rule and
// the runner-row anonymization sweep above.
func sweepNonFinishedRegistrations(ctx context.Context, s store.Store, now time.Time) error {
	threshold := now.AddDate(0, -nonFinishedRegistrationWindow, 0)
	regs, err := s.ListRegistrations(ctx)
	if err != nil {
		return err
	}
	for _, reg := range regs {
		if reg.Status != models.StatusDNF && reg.Status != models.StatusDNS {
			continue
		}
		race, err := s.GetRace(ctx, reg.RaceID)
		if err != nil || race == nil {
			continue
		}
		event, err := s.GetEvent(ctx, race.EventID)
		if err != nil || event == nil {
			continue
		}
		raceDate, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			continue
		}
		if raceDate.After(threshold) {
			continue
		}
		if err := s.DeleteRegistration(ctx, reg.RaceID, reg.RunnerID); err != nil {
			log.Printf("retention: delete %s/%s registration failed: %v", reg.RaceID, reg.RunnerID, err)
			continue
		}
		log.Printf("retention: dropped %s registration race=%s runner=%s raceDate=%s",
			reg.Status, reg.RaceID, reg.RunnerID, event.Date)
	}
	return nil
}

func sweepRunners(ctx context.Context, s store.Store, now time.Time) error {
	runners, err := s.ListRunners(ctx)
	if err != nil {
		return err
	}
	for _, r := range runners {
		if r.DeletionPendingUntil == nil {
			continue
		}
		if r.DeletionPendingUntil.After(now) {
			continue
		}
		if err := purgeRunner(ctx, s, r); err != nil {
			log.Printf("retention: purge runner id=%s failed: %v", r.ID, err)
			continue
		}
		log.Printf("retention: purged runner id=%s accountId=%s", r.ID, r.AccountID)
	}
	return nil
}

func sweepAccounts(ctx context.Context, s store.Store, now time.Time) error {
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if a.DeletionPendingUntil == nil {
			continue
		}
		if a.DeletionPendingUntil.After(now) {
			continue
		}
		if err := purgeAccount(ctx, s, a); err != nil {
			log.Printf("retention: purge account id=%s failed: %v", a.ID, err)
			continue
		}
		log.Printf("retention: purged account id=%s", a.ID)
	}
	return nil
}

// purgeRunner anonymizes a single Runner. Drops non-finished registrations.
// Finished registrations are kept verbatim — they'll render as "Anonym" via
// the anonymized runner row when the leaderboard is queried.
func purgeRunner(ctx context.Context, s store.Store, r models.Runner) error {
	regs, err := s.ListRegistrationsByRunner(ctx, r.ID)
	if err != nil {
		return err
	}
	for _, reg := range regs {
		if reg.Status == models.StatusFinished {
			continue
		}
		if err := s.DeleteRegistration(ctx, reg.RaceID, reg.RunnerID); err != nil {
			return err
		}
	}

	// Hard-overwrite PII fields. Keep ID + AccountID so the finished
	// registrations stay linked to *some* row (anonymized name).
	anonymized := models.Runner{
		ID:        r.ID,
		AccountID: r.AccountID,
		Name:      "Anonym",
		BirthDate: "",
		Gender:    "",
		Consents:  models.RunnerConsents{}, // wiped — there's no PII left to consent over
		CreatedAt: r.CreatedAt,
		// DeletionPendingUntil deliberately omitted (nil) — sweep is done.
	}
	return s.UpdateRunner(ctx, anonymized)
}

// purgeAccount nulls out PII on the account row, anonymizes every owned
// runner, then deletes the account row + email sentinel. We do it in this
// order so a mid-sweep crash leaves runners safely anonymized; the account
// row still standing just means the next sweep retries the final delete.
func purgeAccount(ctx context.Context, s store.Store, a models.Account) error {
	runners, err := s.ListRunnersByAccount(ctx, a.ID)
	if err != nil {
		return err
	}
	for _, r := range runners {
		// Even if the runner wasn't individually pending-deletion, the
		// account-level erase cascades to it. Purge anyway.
		if err := purgeRunner(ctx, s, r); err != nil {
			return err
		}
	}
	// Drop the row entirely — including the EMAIL# sentinel so the email
	// can theoretically be reused (or stay free).
	if err := s.DeleteAccount(ctx, a.ID, a.Email); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// sweepInactiveAccounts marks any account that hasn't seen activity in the
// last 36 months as pending-deletion (30-day grace window), mints a restore
// token, and emails an inactivity notice to the account email.
//
// "Activity" = any of: account createdAt, lastLoginAt (any /dsr/verify),
// any registration createdAt on a runner the account owns, any finished
// race for one of those runners. The most recent of these is the
// activity timestamp; if older than threshold, the account is swept.
//
// Admin accounts and accounts already in pending-deletion are skipped.
// Idempotent: once DeletionPendingUntil is set, the standard sweepAccounts
// path takes over.
func sweepInactiveAccounts(
	ctx context.Context,
	s store.Store,
	recorder audit.Recorder,
	renderer *email.Renderer,
	now time.Time,
) error {
	threshold := now.AddDate(0, -accountInactivityWindow, 0)
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if a.IsAdmin {
			continue
		}
		if a.DeletionPendingUntil != nil {
			continue
		}
		runners, err := s.ListRunnersByAccount(ctx, a.ID)
		if err != nil {
			log.Printf("retention: inactivity list runners account=%s failed: %v", a.ID, err)
			continue
		}
		latest := mostRecentAccountActivity(ctx, s, a, runners)
		if latest.After(threshold) {
			continue
		}
		if err := softDeleteAccount(ctx, s, a, runners, now); err != nil {
			log.Printf("retention: inactivity soft-delete account=%s failed: %v", a.ID, err)
			continue
		}
		until := now.Add(inactivityDeletionGrace)
		if err := issueInactivityRestoreLink(ctx, s, renderer, a, until); err != nil {
			// Don't roll back the soft-delete — the user gets paged via
			// their next /my-data login (we already wrote the audit row
			// they can see there), and the next sweep won't re-page
			// because DeletionPendingUntil is set.
			log.Printf("retention: inactivity email/token account=%s failed: %v", a.ID, err)
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  a.ID,
			At:         now,
			Action:     "retention.account.inactivity",
			Actor:      "system",
			TargetType: "account",
			TargetID:   a.ID,
			Summary:    "Scheduled for deletion after 36 months of inactivity",
		})
		log.Printf("retention: inactivity soft-delete account=%s lastActivity=%s deletionAt=%s",
			a.ID, latest.Format("2006-01-02"), until.Format("2006-01-02"))
	}
	return nil
}

// mostRecentAccountActivity returns the latest "activity" timestamp for an
// account. Mirrors the computation in buildDSRMeBody so the dashboard
// countdown and the sweep agree on when to fire.
func mostRecentAccountActivity(ctx context.Context, s store.Store, a models.Account, runners []models.Runner) time.Time {
	latest := a.CreatedAt
	if a.LastLoginAt != nil && a.LastLoginAt.After(latest) {
		latest = *a.LastLoginAt
	}
	for _, r := range runners {
		regs, err := s.ListRegistrationsByRunner(ctx, r.ID)
		if err != nil {
			continue
		}
		for _, reg := range regs {
			if reg.CreatedAt.After(latest) {
				latest = reg.CreatedAt
			}
			if reg.Status != models.StatusFinished {
				continue
			}
			race, err := s.GetRace(ctx, reg.RaceID)
			if err != nil || race == nil {
				continue
			}
			event, err := s.GetEvent(ctx, race.EventID)
			if err != nil || event == nil {
				continue
			}
			d, err := time.Parse("2006-01-02", event.Date)
			if err != nil {
				continue
			}
			if d.After(latest) {
				latest = d
			}
		}
	}
	return latest
}

// softDeleteAccount flips the account row + each owned runner row into the
// 30-day grace window, and cascades non-finished registrations to
// pending_deletion so they drop out of admin pipelines immediately.
// Finished/DNF/DNS rows are race history and stay put.
func softDeleteAccount(ctx context.Context, s store.Store, a models.Account, runners []models.Runner, now time.Time) error {
	until := now.Add(inactivityDeletionGrace)
	a.DeletionPendingUntil = &until
	if err := s.UpdateAccount(ctx, a); err != nil {
		return err
	}
	for _, r := range runners {
		if r.DeletionPendingUntil != nil {
			continue
		}
		r.DeletionPendingUntil = &until
		if err := s.UpdateRunner(ctx, r); err != nil {
			return err
		}
		if err := cascadeRegistrations(ctx, s, r.ID); err != nil {
			return err
		}
	}
	return nil
}

// cascadeRegistrations is the retention-Lambda copy of
// cascadePendingDeletion in internal/api/dsr.go. Same rules; lives here
// because cmd/retention can't import internal/api.
func cascadeRegistrations(ctx context.Context, s store.Store, runnerID string) error {
	regs, err := s.ListRegistrationsByRunner(ctx, runnerID)
	if err != nil {
		return err
	}
	for _, reg := range regs {
		switch reg.Status {
		case models.StatusFinished, models.StatusDNF, models.StatusDNS, models.StatusPendingDeletion:
			continue
		}
		if err := s.UpdateRegistrationStatus(ctx, reg.ID, models.StatusPendingDeletion); err != nil {
			return err
		}
	}
	return nil
}

// issueInactivityRestoreLink mints a dsr_restore token and emails the
// inactivity notice (different copy from a user-initiated deletion: the
// user didn't ask, we're the ones flagging it).
func issueInactivityRestoreLink(ctx context.Context, s store.Store, renderer *email.Renderer, a models.Account, until time.Time) error {
	tokenID, err := newRetentionTokenID()
	if err != nil {
		return err
	}
	token := models.MagicToken{
		Kind:      models.TokenKindDSRRestore,
		ID:        tokenID,
		AccountID: a.ID,
		ExpiresAt: until,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.CreateMagicToken(ctx, token); err != nil {
		return err
	}
	return sendInactivityNoticeEmail(ctx, renderer, a.Email, a.Locale, tokenID, until)
}

func newRetentionTokenID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// retentionInactivityVars matches the {{.DeletionDate}} / {{.Link}}
// placeholders in the retention-inactivity template.
type retentionInactivityVars struct {
	Link         string
	DeletionDate string
}

// sendInactivityNoticeEmail tells the user we're about to delete their data
// because they haven't logged in / registered / raced in 36 months, and
// offers the same restore-link flow used for user-initiated deletes.
func sendInactivityNoticeEmail(ctx context.Context, renderer *email.Renderer, toEmail, locale, tokenID string, deletionAt time.Time) error {
	base := os.Getenv("SITE_BASE_URL")
	if base == "" {
		base = "https://running.rydback.net"
	}
	link := base + "/my-data/restore?token=" + tokenID
	return renderer.Send(ctx, email.SendOptions{
		Slug:   models.EmailTemplateSlugRetentionInactivity,
		To:     toEmail,
		Locale: locale,
		Vars:   retentionInactivityVars{Link: link, DeletionDate: deletionAt.Format("2006-01-02")},
	})
}
