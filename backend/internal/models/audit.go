package models

import "time"

// AuditRow is one event in the per-account activity log (GDPR A1.1.4 +
// A1.2). The same row stream surfaces both in the user-facing "Your
// activity" view on /my-data and in any future admin audit dashboard.
//
// Action is a stable machine-readable code (e.g. "consent.marketing.update",
// "email.sent", "runner.erased") that the frontend maps to a localized
// label. Summary is a short human-readable string; for action codes the
// frontend doesn't know about it's the only thing that renders.
//
// Actor identifies who triggered the action:
//   - "user"       — the account holder via /my-data
//   - "admin:<id>" — an admin via the admin console
//   - "system"     — automated (retention sweep, scheduled job)
//
// TargetType + TargetID locate the affected entity (e.g. "runner" + UUID).
// Diff is an optional JSON-ish summary of what changed; omitted for actions
// where summary alone is enough.
type AuditRow struct {
	AccountID  string
	At         time.Time
	Action     string
	Actor      string
	TargetType string
	TargetID   string
	Summary    string
	Diff       string
}
