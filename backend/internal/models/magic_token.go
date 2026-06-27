package models

import "time"

// TokenKind discriminates between the magic-link kinds we issue. Each kind
// has its own TTL, its own email template, and its own meaning for ContextID.
type TokenKind string

const (
	// TokenKindGuardian — parental-consent click for an under-13 registration.
	// ContextID = registrationId. Lifetime: 7d.
	TokenKindGuardian TokenKind = "guardian"

	// TokenKindDSRAccess — issued by POST /dsr/request-link when the email
	// matches an active account; redeemed to start a /my-data session.
	// ContextID = "". Lifetime: 30m (one-time use, short window).
	TokenKindDSRAccess TokenKind = "dsr_access"

	// TokenKindDSRRestore — issued when /dsr/request-link targets an account
	// in pending-deletion; redeemed to clear DeletionPendingUntil.
	// ContextID = "". Lifetime: 30d (matches the grace window).
	TokenKindDSRRestore TokenKind = "dsr_restore"

	// TokenKindEmailChange — confirmation link sent to the NEW email when a
	// user requests an email change. Redeeming swaps the EMAIL# sentinel and
	// updates account.email atomically. ContextID = newNormalizedEmail.
	// Lifetime: 1d.
	TokenKindEmailChange TokenKind = "email_change"
)

// MagicToken is the database-backed proof-of-click for any one-shot link we
// email out (parental consent, DSR access/restore, email-change confirmation).
// Stored in the-run-magic-tokens. Auto-purged via DynamoDB TTL on ExpiresAt.
//
// The ID is the random opaque string that lands in the URL. Stored unhashed:
// anyone with DB read access could already perform the linked action directly,
// so hashing wouldn't change the threat model. ContextID carries kind-specific
// data (registrationId for guardian, newEmail for email_change, etc.).
type MagicToken struct {
	Kind      TokenKind
	ID        string `gdpr:"credential;purposes=dsr,registration"`
	AccountID string
	ContextID string `gdpr:"operational;purposes=dsr,registration"`
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
