package email

import (
	"github.com/BirgerRydback/the-run/backend/internal/models"
)

// SeedTemplate is the in-code default for one email template. Used by
// SeededTemplate (renderer fallback when DynamoDB has no published row) and by
// the seed routine that populates DynamoDB on first boot.
type SeedTemplate struct {
	Slug               models.EmailTemplateSlug
	DisplayName        string
	SubjectSv          string
	BodySv             string
	SubjectEn          string
	BodyEn             string
	AvailableVariables []string
}

// templateSeeds carries the canonical default body for every slug in
// models.KnownEmailTemplateSlugs. The Swedish bodies are exactly the strings
// that lived in the handler functions before the renderer migration; English
// versions are direct translations so the first admin who logs in sees both
// languages already filled in.
//
// Bodies use Go text/template syntax: `{{.FieldName}}` substitutes the named
// field of the variables struct passed to Renderer.Send. AvailableVariables
// documents the expected names for the admin UI's reference panel; the
// renderer itself ignores it (template execution validates by parsing).
var templateSeeds = map[models.EmailTemplateSlug]SeedTemplate{
	models.EmailTemplateSlugRegistrationConfirmation: {
		Slug:        models.EmailTemplateSlugRegistrationConfirmation,
		DisplayName: "Race registration confirmation",
		SubjectSv:   "Anmälan bekräftad: {{.EventName}} — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"Tack — vi har tagit emot din anmälan till {{.EventName}}.\n\n" +
			"Här är detaljerna:\n\n" +
			"  • Löpare: {{.RunnerName}}\n" +
			"  • Klass: {{.RaceName}} ({{.RaceDistance}})\n" +
			"  • Datum: {{.EventDate}}\n" +
			"  • Plats: {{.EventLocation}}\n\n" +
			"{{if gt .RegistrationFeeOre 0}}Anmälningsavgift: {{.RegistrationFeeFormatted}}\n\n" +
			"Din anmälan är preliminär tills vi har bekräftat din betalning. Betala via Swish eller bankgiro enligt instruktionerna på vår hemsida — så snart vi prickat av betalningen är du klar för start.\n\n" +
			"{{end}}" +
			"Vi mejlar dig igen närmare loppet med startnummer och praktisk information.\n\n" +
			"Behöver du uppdatera dina uppgifter eller ångra anmälan? Du hanterar allt själv på:\n\n" +
			"{{.ManageDataUrl}}\n\n" +
			"Vi ses på startlinjen!\n",
		SubjectEn: "Registration confirmed: {{.EventName}} — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"Thanks — we've received your registration for {{.EventName}}.\n\n" +
			"Here are the details:\n\n" +
			"  • Runner: {{.RunnerName}}\n" +
			"  • Race: {{.RaceName}} ({{.RaceDistance}})\n" +
			"  • Date: {{.EventDate}}\n" +
			"  • Location: {{.EventLocation}}\n\n" +
			"{{if gt .RegistrationFeeOre 0}}Registration fee: {{.RegistrationFeeFormatted}}\n\n" +
			"Your registration is provisional until we've confirmed your payment. Pay via Swish or bank transfer following the instructions on our website — as soon as we tick off your payment you're good to go.\n\n" +
			"{{end}}" +
			"We'll email you again closer to race day with your bib number and practical info.\n\n" +
			"Need to update your details or cancel? You can manage everything yourself at:\n\n" +
			"{{.ManageDataUrl}}\n\n" +
			"See you at the start line!\n",
		AvailableVariables: []string{"RunnerName", "EventName", "EventDate", "EventLocation", "RaceName", "RaceDistance", "ManageDataUrl", "RegistrationFeeOre", "RegistrationFeeFormatted"},
	},
	models.EmailTemplateSlugGuardianConsent: {
		Slug:        models.EmailTemplateSlugGuardianConsent,
		DisplayName: "Guardian consent (under-13 registration)",
		SubjectSv:   "Bekräfta målsmans samtycke — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"Du har registrerat {{.RunnerName}} till ett lopp på Ingmarsöloppet. Eftersom hen är under 13 år behöver vi en målsmans samtycke innan anmälan blir aktiv.\n\n" +
			"Klicka på länken nedan för att bekräfta inom 7 dagar — annars förfaller anmälan automatiskt:\n\n" +
			"{{.Link}}\n\n" +
			"Om du inte gjort den här anmälan kan du bortse från det här mejlet; ingenting registreras förrän du klickat på länken.\n",
		SubjectEn: "Confirm guardian consent — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"You have registered {{.RunnerName}} for a race at Ingmarsöloppet. Because they are under 13, we need a guardian's consent before the registration becomes active.\n\n" +
			"Click the link below to confirm within 7 days — otherwise the registration will expire automatically:\n\n" +
			"{{.Link}}\n\n" +
			"If you did not make this registration you can ignore this email; nothing is recorded until you click the link.\n",
		AvailableVariables: []string{"RunnerName", "Link"},
	},
	models.EmailTemplateSlugDSRAccess: {
		Slug:        models.EmailTemplateSlugDSRAccess,
		DisplayName: "My-data magic link",
		SubjectSv:   "Hantera dina uppgifter — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"Du har begärt att hantera dina uppgifter hos Ingmarsöloppet. Klicka på länken nedan inom 30 minuter för att logga in:\n\n" +
			"{{.Link}}\n\n" +
			"Om du inte begärt det här mejlet kan du bortse från det.\n",
		SubjectEn: "Manage your data — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"You have requested to manage your data at Ingmarsöloppet. Click the link below within 30 minutes to sign in:\n\n" +
			"{{.Link}}\n\n" +
			"If you did not request this email, you can ignore it.\n",
		AvailableVariables: []string{"Link"},
	},
	models.EmailTemplateSlugDSREmailChange: {
		Slug:        models.EmailTemplateSlugDSREmailChange,
		DisplayName: "Email-change confirmation",
		SubjectSv:   "Bekräfta din nya e-postadress — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"En begäran har gjorts att byta e-postadress till den här adressen på Ingmarsöloppet. Klicka på länken nedan inom 24 timmar för att bekräfta:\n\n" +
			"{{.Link}}\n\n" +
			"Om du inte gjort den här begäran kan du bortse från mejlet — inget händer förrän du klickat på länken.\n",
		SubjectEn: "Confirm your new email address — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"A request has been made to change the email address on an Ingmarsöloppet account to this address. Click the link below within 24 hours to confirm:\n\n" +
			"{{.Link}}\n\n" +
			"If you did not make this request, you can ignore this email — nothing happens until you click the link.\n",
		AvailableVariables: []string{"Link"},
	},
	models.EmailTemplateSlugDSRRestore: {
		Slug:        models.EmailTemplateSlugDSRRestore,
		DisplayName: "Account deletion grace-period restore",
		SubjectSv:   "Dina uppgifter raderas snart — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"Vi har tagit emot din begäran om att radera dina uppgifter hos Ingmarsöloppet.\n\n" +
			"Uppgifterna raderas slutgiltigt den {{.DeletionDate}}. Fram till dess är de dolda men kan återställas.\n\n" +
			"Klicka på länken nedan om du ångrar dig och vill behålla dina uppgifter:\n\n" +
			"{{.Link}}\n\n" +
			"Behöver du ingen ångerlucka? Gör inget — uppgifterna raderas automatiskt på utsatt datum.\n",
		SubjectEn: "Your data will be deleted soon — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"We have received your request to delete your data at Ingmarsöloppet.\n\n" +
			"Your data will be permanently deleted on {{.DeletionDate}}. Until then it is hidden but can be restored.\n\n" +
			"Click the link below if you change your mind and want to keep your data:\n\n" +
			"{{.Link}}\n\n" +
			"Don't need to undo? Do nothing — your data will be deleted automatically on the date above.\n",
		AvailableVariables: []string{"DeletionDate", "Link"},
	},
	models.EmailTemplateSlugRetentionInactivity: {
		Slug:        models.EmailTemplateSlugRetentionInactivity,
		DisplayName: "Inactive-account deletion notice",
		SubjectSv:   "Inaktivt konto — dina uppgifter raderas snart — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"Du har inte använt ditt konto hos Ingmarsöloppet på över 36 månader. Enligt vår dataskyddspolicy tar vi automatiskt bort konton som varit inaktiva så länge.\n\n" +
			"Dina uppgifter raderas slutgiltigt den {{.DeletionDate}}. Fram till dess är de dolda men kan återställas.\n\n" +
			"Klicka på länken nedan om du vill behålla ditt konto:\n\n" +
			"{{.Link}}\n\n" +
			"Vill du inte behålla det? Gör inget — uppgifterna raderas automatiskt på utsatt datum.\n",
		SubjectEn: "Inactive account — your data will be deleted soon — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"You have not used your Ingmarsöloppet account in over 36 months. Per our data-protection policy we automatically remove accounts that have been inactive for this long.\n\n" +
			"Your data will be permanently deleted on {{.DeletionDate}}. Until then it is hidden but can be restored.\n\n" +
			"Click the link below if you want to keep your account:\n\n" +
			"{{.Link}}\n\n" +
			"Don't want to keep it? Do nothing — your data will be deleted automatically on the date above.\n",
		AvailableVariables: []string{"DeletionDate", "Link"},
	},
	models.EmailTemplateSlugAdminInvite: {
		Slug:        models.EmailTemplateSlugAdminInvite,
		DisplayName: "Admin invitation",
		SubjectSv:   "Du har blivit inbjuden som administratör — Ingmarsöloppet",
		BodySv: "Hej!\n\n" +
			"{{.InvitedByMail}} har bjudit in dig som administratör för Ingmarsöloppet.\n\n" +
			"Klicka på länken nedan inom 7 dagar för att skapa ett lösenord och aktivera kontot:\n\n" +
			"{{.Link}}\n\n" +
			"Vid första inloggningen får du registrera tvåfaktorsautentisering (TOTP) med en autentiseringsapp.\n\n" +
			"Om du inte väntat dig den här inbjudan kan du bortse från mejlet.\n",
		SubjectEn: "You've been invited as an administrator — Ingmarsöloppet",
		BodyEn: "Hello!\n\n" +
			"{{.InvitedByMail}} has invited you to become an administrator of Ingmarsöloppet.\n\n" +
			"Click the link below within 7 days to set a password and activate your account:\n\n" +
			"{{.Link}}\n\n" +
			"On first sign-in you will enroll two-factor authentication (TOTP) with an authenticator app.\n\n" +
			"If you weren't expecting this invitation you can ignore this email.\n",
		AvailableVariables: []string{"InvitedByMail", "Link"},
	},
}

// SeededTemplate returns the canonical default for a slug. Used by the
// renderer when DynamoDB has no published row yet (initial deploy, or after a
// destroy/recreate of the LocalStack tables) and by the seed routine to
// populate the database.
func SeededTemplate(slug models.EmailTemplateSlug) (SeedTemplate, bool) {
	t, ok := templateSeeds[slug]
	return t, ok
}

// AllSeededTemplates returns every seed in code order. Used by the seed
// routine that populates DynamoDB on first boot.
func AllSeededTemplates() []SeedTemplate {
	out := make([]SeedTemplate, 0, len(models.KnownEmailTemplateSlugs))
	for _, slug := range models.KnownEmailTemplateSlugs {
		if t, ok := templateSeeds[slug]; ok {
			out = append(out, t)
		}
	}
	return out
}
