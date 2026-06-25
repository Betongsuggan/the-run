package models

import "time"

// GuardianToken is the database-backed proof-of-click for a parental-consent
// magic link (GDPR A0.4). One row per under-13 registration awaiting consent.
// Revocable by deleting the row before the guardian clicks. Auto-purged via
// DynamoDB TTL on ExpiresAt so abandoned tokens don't accumulate.
//
// The ID is the random opaque string that lands in the magic-link URL. It's
// stored unhashed: anyone with DB read access could already flip the linked
// registration's status directly, so hashing wouldn't change the threat model
// in any meaningful way.
type GuardianToken struct {
	ID                string
	RegistrationID    string
	GuardianAccountID string
	ExpiresAt         time.Time
	UsedAt            *time.Time
	CreatedAt         time.Time
}
