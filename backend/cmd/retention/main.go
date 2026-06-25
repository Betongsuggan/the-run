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
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

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
	now := time.Now().UTC()

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
