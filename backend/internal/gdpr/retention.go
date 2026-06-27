// Package gdpr is the single source of truth for the retention windows,
// processing-purpose registry, sub-processor list, and table-storage posture
// that the ROPA/DPIA generator reads. Every retention duration the code
// honours — sweep windows, magic-token TTLs, session lifetimes — lives here.
// Importers in cmd/retention, internal/api, and internal/auth pull values
// from this package so the legal record and the runtime behaviour can't drift.
package gdpr

import "time"

// Month-granularity windows. The retention sweep iterates by calendar month
// (via time.AddDate(0, -N, 0)) so storing these as ints rather than
// time.Durations keeps the call sites readable.
const (
	// NonConsentingRunnerWindowMonths anonymizes a runner's PII this long
	// after their most recent finished race when they opted out of public
	// results. Sliding: any new finished race resets the clock.
	NonConsentingRunnerWindowMonths = 36

	// NonFinishedRegistrationMonths drops DNS/DNF registrations this long
	// after race day. They aren't result history; lingering rows just
	// add PII surface area.
	NonFinishedRegistrationMonths = 6

	// AuditLogMonths is the retention ceiling on admin-audit rows.
	// Matches B1.4's 24-month bound.
	AuditLogMonths = 24

	// AccountInactivityMonths soft-deletes accounts that haven't logged
	// in, registered, or raced for this long. Sliding: any activity
	// resets the clock.
	AccountInactivityMonths = 36
)

// Duration-granularity windows. These all back things that are either
// DynamoDB TTL attributes (auth-attempts, magic-tokens) or in-memory /
// signed-cookie lifetimes.
const (
	// InactivityDeletionGrace is the runway between an inactivity notice
	// email and the hard purge. Long enough for the user to act, short
	// enough not to keep stale PII around.
	InactivityDeletionGrace = 30 * 24 * time.Hour

	// SessionTTL is the admin session cookie's absolute lifetime.
	SessionTTL = 8 * time.Hour

	// ChallengeTTL is the short cookie that carries password→TOTP step
	// state between the two halves of admin login.
	ChallengeTTL = 5 * time.Minute

	// AuthAttemptTTL bounds how long a failed login is counted against
	// the per-account lockout threshold.
	AuthAttemptTTL = 15 * time.Minute

	// DSRAccessTokenTTL is the one-shot magic-link window for /my-data
	// initial login.
	DSRAccessTokenTTL = 30 * time.Minute

	// DSRSessionTTL is the /my-data session cookie's absolute lifetime
	// after the magic link has been redeemed.
	DSRSessionTTL = 2 * time.Hour

	// EmailChangeTokenTTL is the window in which a user can confirm a new
	// email address via the link sent to that address.
	EmailChangeTokenTTL = 24 * time.Hour

	// GuardianTokenTTL is how long a parental-consent magic link stays
	// valid (GDPR A0.4).
	GuardianTokenTTL = 7 * 24 * time.Hour
)

// Retention is one row in the human-readable retention registry. The ROPA
// generator references entries by key (set on each Purpose); the descriptions
// are what land in docs/gdpr/ropa.md.
type Retention struct {
	Key      string
	DescSv   string
	DescEn   string
	Sliding  bool // true when activity resets the clock
	Months   int  // 0 if the window is expressed as Duration
	Duration time.Duration
}

// Retentions is the keyed registry used by the generator + ROPA. Each row
// renders as the "Lagringstid" line in a processing-activity section.
var Retentions = map[string]Retention{
	"runner-sliding-36mo": {
		Key:     "runner-sliding-36mo",
		DescSv:  "36 månader rullande från senaste loppdeltagande. Klockan börjar om vid varje nytt lopp.",
		DescEn:  "36 months sliding from the most recent race participation. The clock resets on each new race.",
		Sliding: true,
		Months:  NonConsentingRunnerWindowMonths,
	},
	"runner-indefinite-if-consenting": {
		Key:    "runner-indefinite-if-consenting",
		DescSv: "Tills vidare om publika resultat tillåts (löparhistorik).",
		DescEn: "Kept indefinitely if public-results consent is granted (race history).",
	},
	"dns-no-show-6mo": {
		Key:    "dns-no-show-6mo",
		DescSv: "6 månader efter loppet (anmälningar utan start).",
		DescEn: "6 months after the race (registrations without start).",
		Months: NonFinishedRegistrationMonths,
	},
	"account-inactivity-36mo": {
		Key:     "account-inactivity-36mo",
		DescSv:  "36 månader rullande från senaste aktivitet (inloggning, anmälan eller lopp).",
		DescEn:  "36 months sliding from the most recent activity (login, registration, or race).",
		Sliding: true,
		Months:  AccountInactivityMonths,
	},
	"audit-24mo": {
		Key:    "audit-24mo",
		DescSv: "24 månader.",
		DescEn: "24 months.",
		Months: AuditLogMonths,
	},
	"consent-until-withdrawn": {
		Key:    "consent-until-withdrawn",
		DescSv: "Tills samtycket återkallas (avregistrering från nyhetsbrevet).",
		DescEn: "Until consent is withdrawn (newsletter unsubscribe).",
	},
	"session-8h": {
		Key:      "session-8h",
		DescSv:   "8 timmar (absolut livslängd för adminsession).",
		DescEn:   "8 hours (admin session absolute lifetime).",
		Duration: SessionTTL,
	},
	"auth-attempt-15min": {
		Key:      "auth-attempt-15min",
		DescSv:   "15 minuter (misslyckade inloggningsförsök).",
		DescEn:   "15 minutes (failed login attempts).",
		Duration: AuthAttemptTTL,
	},
	"magic-token-short": {
		Key:    "magic-token-short",
		DescSv: "Kort engångslänk (30 minuter – 7 dagar beroende på syfte).",
		DescEn: "Short-lived one-shot link (30 minutes – 7 days depending on purpose).",
	},
}
