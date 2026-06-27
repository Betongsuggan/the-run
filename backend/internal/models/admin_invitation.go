package models

import "time"

// AdminInvitation is a pending invite issued by an existing admin to onboard
// a new admin by email. The raw invitation token only lives in the magic link
// emailed to the recipient; the store row carries the SHA-256 hash so a stolen
// DB dump can't be replayed against the accept endpoint.
//
// The row is consumed (UsedAt set) atomically with creating the new Account in
// /admin/accept. Expired or revoked rows are kept until DynamoDB TTL evicts
// them so the accept endpoint can produce a precise "expired" / "already
// used" error instead of a bare 404.
type AdminInvitation struct {
	// TokenHash is the SHA-256 hex of the raw token issued in the email.
	// Used as the primary key so lookups don't need a GSI.
	TokenHash string

	Email         string `gdpr:"contact;purposes=admin-invite;subject=Administratör"`
	Locale        string // "sv" or "en" — picked by the inviter, sets the new account's Locale.
	InvitedByID   string
	InvitedByMail string `gdpr:"contact;purposes=admin-invite;subject=Administratör"`

	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
