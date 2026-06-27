package gdpr

// TablePosture mirrors a single dynamodb.NewTable call in
// infra/database/database.go. The storage-consistency test (see
// storage_consistency_test.go) AST-walks the Pulumi file and verifies that
// every NewTable call has a matching entry here with the same SSE/PITR/TTL
// posture — so this registry can't drift silently from the actual infra.
//
// Used by the generator to emit the "Tekniska och organisatoriska
// säkerhetsåtgärder" section of the ROPA and the table-by-table retention
// table.
type TablePosture struct {
	Name         string
	SSE          bool
	PITR         bool
	TTLAttribute string // empty if no DynamoDB TTL
	DescSv       string
	DescEn       string
}

var Tables = []TablePosture{
	{
		Name:   "the-run-runners",
		SSE:    true,
		PITR:   true,
		DescSv: "Löparuppgifter (namn, födelsedatum, kön, samtycke).",
		DescEn: "Runner records (name, date of birth, gender, consent).",
	},
	{
		Name:   "the-run-registrations",
		SSE:    true,
		PITR:   true,
		DescSv: "Anmälningar och resultat per löpare och klass.",
		DescEn: "Registrations and results per runner and race.",
	},
	{
		Name:   "the-run-accounts",
		SSE:    true,
		PITR:   true,
		DescSv: "Konton (e-post, lösenord, MFA, marknadssamtycke).",
		DescEn: "Accounts (email, password hash, MFA, marketing consent).",
	},
	{
		Name:   "the-run-events",
		SSE:    true,
		PITR:   false,
		DescSv: "Loppen — namn, år, datum, plats.",
		DescEn: "Events — name, year, date, location.",
	},
	{
		Name:   "the-run-races",
		SSE:    true,
		PITR:   false,
		DescSv: "Klasser inom varje lopp (5K, 10K, barnens lopp).",
		DescEn: "Races within each event (5K, 10K, kids' race).",
	},
	{
		Name:         "the-run-auth-attempts",
		SSE:          true,
		PITR:         false,
		TTLAttribute: "expiresAt",
		DescSv:       "Misslyckade inloggningsförsök — auto-rensas via TTL.",
		DescEn:       "Failed login attempts — auto-cleared via TTL.",
	},
	{
		Name:         "the-run-magic-tokens",
		SSE:          true,
		PITR:         false,
		TTLAttribute: "expiresAt",
		DescSv:       "Engångslänkar (DSR, målsmanssamtycke, e-poständring) — auto-rensas via TTL.",
		DescEn:       "One-shot magic links (DSR, guardian consent, email change) — auto-cleared via TTL.",
	},
	{
		Name:   "the-run-audit",
		SSE:    true,
		PITR:   false,
		DescSv: "Aktivitetsloggar för admin- och DSR-händelser.",
		DescEn: "Activity log for admin and DSR events.",
	},
	{
		Name:         "the-run-rate-limit",
		SSE:          true,
		PITR:         false,
		TTLAttribute: "expiresAt",
		DescSv:       "Korta IP-räknare för bot-skydd vid anmälan — auto-rensas via TTL.",
		DescEn:       "Short-lived per-IP counters for bot protection — auto-cleared via TTL.",
	},
	{
		Name:   "the-run-policies",
		SSE:    true,
		PITR:   true,
		DescSv: "Versionerade sekretesspolicyer.",
		DescEn: "Versioned privacy policies.",
	},
	{
		Name:   "the-run-policy-revisions",
		SSE:    true,
		PITR:   false,
		DescSv: "Bevarade tidigare versioner av policyer (snapshots).",
		DescEn: "Preserved earlier policy revisions (snapshots).",
	},
	{
		Name:   "the-run-email-templates",
		SSE:    true,
		PITR:   true,
		DescSv: "Versionerade textinnehåll för transaktionsmejl (kvitton, magic links).",
		DescEn: "Versioned plain-text bodies for transactional emails (receipts, magic links).",
	},
	{
		Name:   "the-run-email-template-revisions",
		SSE:    true,
		PITR:   false,
		DescSv: "Bevarade tidigare versioner av mejlmallar (snapshots).",
		DescEn: "Preserved earlier email-template revisions (snapshots).",
	},
	{
		Name:         "the-run-admin-invitations",
		SSE:          true,
		PITR:         false,
		TTLAttribute: "expiresAt",
		DescSv:       "Pågående administratörsinbjudningar (e-post + hashad engångstoken) — auto-rensas via TTL efter 7 dagar.",
		DescEn:       "Pending admin invitations (email + hashed one-shot token) — auto-cleared via TTL after 7 days.",
	},
}

// SecurityMeasures lists the technical/organizational measures the ROPA
// renders under "Säkerhetsåtgärder". Hardcoded here (not derivable from
// struct tags) but lives next to the table registry so they ship together.
type SecurityMeasure struct {
	DescSv string
	DescEn string
}

var SecurityMeasures = []SecurityMeasure{
	{
		DescSv: "Kryptering i vila — AWS-hanterad nyckel på samtliga DynamoDB-tabeller (SSE).",
		DescEn: "Encryption at rest — AWS-managed key on all DynamoDB tables (SSE).",
	},
	{
		DescSv: "Kryptering under överföring — HTTPS via CloudFront och API Gateway.",
		DescEn: "Encryption in transit — HTTPS via CloudFront and API Gateway.",
	},
	{
		DescSv: "Säkerhetskopior — 35-dagars Point-in-Time Recovery på alla PII-bärande tabeller.",
		DescEn: "Backups — 35-day Point-in-Time Recovery on all PII-bearing tables.",
	},
	{
		DescSv: "Adminautentisering — bcrypt (kostnad 12) plus obligatorisk TOTP-MFA.",
		DescEn: "Admin authentication — bcrypt (cost 12) plus mandatory TOTP MFA.",
	},
	{
		DescSv: "Brute-force-skydd — 5 misslyckade försök ger 15 minuters låsning per konto.",
		DescEn: "Brute-force protection — 5 failed attempts trigger a 15-minute lockout per account.",
	},
	{
		DescSv: "Bot- och spamskydd — honeypot-fält och per-IP-throttling på registrering.",
		DescEn: "Bot and spam protection — honeypot field and per-IP throttling on registration.",
	},
	{
		DescSv: "Aktivitetsloggar — 24 månaders retention på admin- och DSR-händelser.",
		DescEn: "Activity logs — 24 months of retention on admin and DSR events.",
	},
	{
		DescSv: "PII utelämnas från applikationsloggar — endast UUID-identifierare loggas.",
		DescEn: "PII is excluded from application logs — only UUID identifiers are logged.",
	},
	{
		DescSv: "Loggretention — 30 dagar i CloudWatch Logs.",
		DescEn: "Log retention — 30 days in CloudWatch Logs.",
	},
}
