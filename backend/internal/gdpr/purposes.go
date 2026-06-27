package gdpr

// LawfulBasis is the GDPR Article 6 legal ground for each processing
// activity. Rendered verbatim into the ROPA's "rättslig grund" line.
type LawfulBasis string

const (
	BasisContract           LawfulBasis = "Avtal (Art. 6.1.b)"
	BasisConsent            LawfulBasis = "Samtycke (Art. 6.1.a)"
	BasisLegitimateInterest LawfulBasis = "Berättigat intresse (Art. 6.1.f)"
	BasisLegalObligation    LawfulBasis = "Rättslig förpliktelse (Art. 6.1.c)"
)

// DataSubject is "who is this person" — runner, guardian of an under-13
// runner, or admin. A single field's effective subjects is the union of
// the subjects on each of its purposes (with an optional per-field
// override via the gdpr struct tag).
type DataSubject string

const (
	SubjectRunner   DataSubject = "Löpare"
	SubjectGuardian DataSubject = "Vårdnadshavare"
	SubjectAdmin    DataSubject = "Administratör"
)

// Category is "what kind of data" — the ROPA's "kategorier av
// personuppgifter" axis. Each PII field's struct tag picks one. Plain
// English variants are kept on the struct for the en-locale path that we
// don't render yet but want to keep wired up.
type Category struct {
	Key    string
	DescSv string
	DescEn string
}

var Categories = map[string]Category{
	"contact": {
		Key:    "contact",
		DescSv: "Kontaktuppgifter",
		DescEn: "Contact data",
	},
	"credential": {
		Key:    "credential",
		DescSv: "Inloggningsuppgifter",
		DescEn: "Authentication credentials",
	},
	"identifying": {
		Key:    "identifying",
		DescSv: "Identifierande uppgifter",
		DescEn: "Identifying data",
	},
	"behavioural": {
		Key:    "behavioural",
		DescSv: "Beteendedata",
		DescEn: "Behavioural data",
	},
	"operational": {
		Key:    "operational",
		DescSv: "Driftsdata",
		DescEn: "Operational data",
	},
	"audit-trail": {
		Key:    "audit-trail",
		DescSv: "Loggdata",
		DescEn: "Audit trail",
	},
}

// Purpose is one processing-activity row in the ROPA. The generator groups
// fields by purpose key (one section per purpose), then resolves subjects +
// retention via the registry. Keys are stable identifiers; the gdpr struct
// tags on model fields reference them.
type Purpose struct {
	Key       string
	DescSv    string
	DescEn    string
	Basis     LawfulBasis
	Subjects  []DataSubject
	Retention string // key into Retentions
}

// OrderedPurposeKeys controls the order purposes appear in the generated
// ROPA. Reading order > alphabetical: start with the primary activity
// (registration), follow into the user-visible ones, then operational.
var OrderedPurposeKeys = []string{
	"registration",
	"public-results",
	"comms",
	"marketing",
	"dsr",
	"admin-auth",
	"fraud-prevention",
	"audit",
}

// OrderedCategoryKeys controls the order categories appear within a single
// purpose section. Contact data first (most identifying), audit-trail last.
var OrderedCategoryKeys = []string{
	"contact",
	"identifying",
	"credential",
	"behavioural",
	"operational",
	"audit-trail",
}

var Purposes = map[string]Purpose{
	"registration": {
		Key:       "registration",
		DescSv:    "Anmälan och löpadministration — startlista, åldersklass, kommunikation om loppet och resultat.",
		DescEn:    "Race registration and operations — start list, age class, race communication, and results.",
		Basis:     BasisContract,
		Subjects:  []DataSubject{SubjectRunner, SubjectGuardian},
		Retention: "runner-sliding-36mo",
	},
	"public-results": {
		Key:       "public-results",
		DescSv:    "Publika resultatlistor — visning av namn, åldersklass och tid på webben (opt-out).",
		DescEn:    "Public result lists — display of name, age class, and time on the web (opt-out).",
		Basis:     BasisLegitimateInterest,
		Subjects:  []DataSubject{SubjectRunner},
		Retention: "runner-indefinite-if-consenting",
	},
	"comms": {
		Key:       "comms",
		DescSv:    "Transaktionsmejl — anmälningsbekräftelser, målsmanslänkar, lösenordsåterställning.",
		DescEn:    "Transactional email — registration confirmations, guardian links, password recovery.",
		Basis:     BasisContract,
		Subjects:  []DataSubject{SubjectRunner, SubjectGuardian, SubjectAdmin},
		Retention: "account-inactivity-36mo",
	},
	"marketing": {
		Key:       "marketing",
		DescSv:    "Nyhetsbrev — information om kommande lopp och relaterade nyheter.",
		DescEn:    "Newsletter — information about upcoming races and related news.",
		Basis:     BasisConsent,
		Subjects:  []DataSubject{SubjectRunner, SubjectGuardian},
		Retention: "consent-until-withdrawn",
	},
	"dsr": {
		Key:       "dsr",
		DescSv:    "Hantering av rättighetsbegäranden — magiclänk-inloggning på /my-data, export, rättelse, radering.",
		DescEn:    "Data-subject-rights handling — magic-link sign-in on /my-data, export, rectification, erasure.",
		Basis:     BasisLegitimateInterest,
		Subjects:  []DataSubject{SubjectRunner, SubjectGuardian, SubjectAdmin},
		Retention: "magic-token-short",
	},
	"admin-auth": {
		Key:       "admin-auth",
		DescSv:    "Administratörsautentisering — lösenord och TOTP-MFA för adminpanelen.",
		DescEn:    "Admin authentication — password and TOTP MFA for the admin panel.",
		Basis:     BasisContract,
		Subjects:  []DataSubject{SubjectAdmin},
		Retention: "session-8h",
	},
	"fraud-prevention": {
		Key:       "fraud-prevention",
		DescSv:    "Bot- och spamskydd — kortvarig IP-räkning vid anmälningsformuläret och inloggningsförsöksspårning.",
		DescEn:    "Bot and spam prevention — short-lived IP counting at the registration form and login-attempt tracking.",
		Basis:     BasisLegitimateInterest,
		Subjects:  []DataSubject{SubjectRunner, SubjectGuardian, SubjectAdmin},
		Retention: "auth-attempt-15min",
	},
	"audit": {
		Key:       "audit",
		DescSv:    "Aktivitetslogg — admin- och DSR-händelser för spårbarhet och utredning av incidenter.",
		DescEn:    "Activity log — admin and DSR events for traceability and incident investigation.",
		Basis:     BasisLegitimateInterest,
		Subjects:  []DataSubject{SubjectRunner, SubjectGuardian, SubjectAdmin},
		Retention: "audit-24mo",
	},
}
