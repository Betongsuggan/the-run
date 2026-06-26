package audit

import (
	"context"
	"log"

	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// AccountLookup resolves a recipient email to an account ID, so the
// wrapped sender can record audit rows under the correct partition key.
// Concretely store.Store satisfies this — but the interface keeps the
// audit package free of store dependency.
type AccountLookup interface {
	GetAccountByEmail(ctx context.Context, email string) (*models.Account, error)
}

// WrapSender returns an email.Sender that records an audit row per send,
// keyed on the recipient's accountId. The underlying sender is called
// regardless of whether the audit row records successfully — email
// delivery is the primary action, audit is best-effort observability.
//
// If the recipient email doesn't resolve to an account (e.g. SES sandbox
// recipients we haven't onboarded, system-to-system mail) the audit row
// is dropped silently — we have no partition key to file it under.
func WrapSender(inner email.Sender, recorder Recorder, lookup AccountLookup) email.Sender {
	return &auditingSender{inner: inner, recorder: recorder, lookup: lookup}
}

type auditingSender struct {
	inner    email.Sender
	recorder Recorder
	lookup   AccountLookup
}

func (s *auditingSender) Send(ctx context.Context, msg email.Message) error {
	// Send first — audit is observability, not gating.
	sendErr := s.inner.Send(ctx, msg)
	if sendErr != nil {
		// Audit the failure too so the user's activity log mentions the
		// attempt. Surfaces "we tried to email you but failed" rather
		// than an unexplained silence.
		s.recordSend(ctx, msg, "email.send.failed")
		return sendErr
	}
	s.recordSend(ctx, msg, "email.sent")
	return nil
}

func (s *auditingSender) recordSend(ctx context.Context, msg email.Message, action string) {
	account, err := s.lookup.GetAccountByEmail(ctx, msg.To)
	if err != nil {
		log.Printf("audit: email recipient lookup failed: to=%s err=%v", msg.To, err)
		return
	}
	if account == nil {
		// No account for this recipient — nothing to attribute. Common
		// during SES sandbox testing.
		return
	}
	s.recorder.Record(ctx, models.AuditRow{
		AccountID:  account.ID,
		Action:     action,
		Actor:      "system",
		TargetType: "email",
		Summary:    msg.Subject,
	})
}
