import type { Gender } from '$lib/types';

export type Locale = 'sv' | 'en';

export const locales: ReadonlyArray<{ code: Locale; name: string }> = [
	{ code: 'sv', name: 'Svenska' },
	{ code: 'en', name: 'English' }
];

const sv = {
	nav: {
		brand: 'Ingmarsöloppet',
		home: 'Hem',
		runners: 'Löpare',
		register: 'Anmäl dig',
		myData: 'Mina uppgifter',
		language: 'Språk',
		theme: 'Tema',
		themeLight: 'Ljust',
		themeDark: 'Mörkt',
		themeSystem: 'System'
	},
	landing: {
		description: 'En plattform för att visa löparresultat från årliga lopp.',
		heroTagline:
			'Löparglädje i Stockholms skärgård — resultat, statistik och historik från varje upplaga.',
		ctaEvents: 'Se loppen',
		ctaRunners: 'Bläddra bland löpare',
		ctaRegister: 'Anmäl dig till ett lopp',
		mockTitle: 'Mockdata-läge',
		mockDescription:
			'Sidorna nedan visar handgjorda exempel. Backend ansluts när UI-formen är fastställd.',
		eventsHeading: 'Lopp',
		runnersHeading: 'Löpare',
		seeAll: 'Visa alla →',
		apiStatus: 'API-status (sanity-test)',
		loading: 'Laddar…',
		error: (msg: string) => `Fel: ${msg}`
	},
	runnersList: {
		heading: 'Löpare',
		description: 'Klicka på en löpare för att se deras resultat.',
		searchPlaceholder: 'Sök på namn…',
		searchLabel: 'Sök löpare',
		resultCount: (n: number) => (n === 1 ? '1 löpare' : `${n} löpare`),
		noMatch: (q: string) => `Ingen löpare matchar "${q}".`
	},
	runner: {
		loading: 'Laddar…',
		notFound: 'Löpare hittades inte.',
		backToRunners: 'Tillbaka till löpare',
		bornIn: (year: number) => `född ${year}`,
		paceOverTime: 'Tempo över tid (endast löpning)',
		distanceDistribution: 'Distansfördelning',
		resultsHeading: 'Resultat'
	},
	result: {
		loading: 'Laddar…',
		notFound: 'Resultat hittades inte.',
		backToRunners: 'Tillbaka till löpare',
		distance: 'Distans',
		finish: 'Måltid',
		pace: 'Tempo',
		placement: 'Placering',
		placementCategory: (place: number, group: string) => `#${place} i ${group}`,
		bib: 'Startnummer',
		category: 'Kategori',
		splitsHeading: 'Mellantider',
		splitsBarTitle: 'Tempo per km',
		fastestKmLabel: 'snabbast',
		conditionsHeading: 'Förhållanden',
		notesHeading: 'Anteckningar'
	},
	illustrations: {
		decorative: 'Dekoration'
	},
	resultsTable: {
		filter: 'Filter:',
		columnDate: 'Datum',
		columnEventRace: 'Lopp / Klass',
		columnDistance: 'Distans',
		columnTime: 'Tid',
		columnPace: 'Tempo',
		columnPlace: 'Plats',
		columnBib: 'Startnr',
		noMatch: 'Inga resultat matchar filtret.'
	},
	event: {
		loading: 'Laddar…',
		notFound: 'Loppet hittades inte.',
		backHome: 'Tillbaka till start',
		racesHeading: 'Lopp',
		participants: (count: number) => (count === 1 ? '1 deltagare' : `${count} deltagare`),
		noResults: 'Inga resultat ännu.'
	},
	race: {
		kids: 'Barnlopp'
	},
	category: {
		gender: { M: 'M', F: 'D', X: 'X' } as Record<Gender, string>
	},
	leaderboard: {
		columnRank: 'Plats',
		columnRunner: 'Löpare',
		columnTime: 'Tid',
		columnPace: 'Tempo',
		columnCategory: 'Kategori',
		columnBib: 'Startnr'
	},
	discipline: {
		all: 'alla',
		run: 'löpning',
		walk: 'gång',
		kids: 'barn'
	},
	summary: {
		races: 'Lopp',
		acrossEvents: (count: number) => (count === 1 ? 'över 1 event' : `över ${count} event`),
		totalDistance: 'Total distans',
		personalBests: 'Personliga rekord (löpning)'
	},
	splits: {
		none: 'Inga mellantider sparade.',
		km: 'Km',
		split: 'Mellantid',
		cumulative: 'Totalt',
		fastest: (km: number, time: string) => `Snabbaste km: km ${km} på ${time}.`
	},
	chart: {
		paceHelp: 'Lägre är snabbare. Håll muspekaren över punkterna för detaljer.',
		noRunsYet: 'Inga löpresultat än.',
		noResultsYet: 'Inga resultat än.'
	},
	register: {
		heading: 'Anmäl dig till loppet',
		subheading: 'Fyll i dina uppgifter och välj vilket lopp du vill springa.',
		nameLabel: 'Namn',
		namePlaceholder: 't.ex. Anna Andersson',
		emailLabel: 'E-post',
		emailPlaceholder: 'din@email.se',
		emailHelp: 'Används för bekräftelse, information om loppet och för att hantera dina uppgifter.',
		dobLabel: 'Födelsedatum',
		genderLabel: 'Kön',
		raceLabel: 'Lopp',
		racePlaceholder: 'Välj ett kommande lopp',
		consentPublicResultsLabel: 'Visa mitt resultat publikt',
		consentPublicResultsHelp:
			'Ditt namn, klass och tid visas i de publika resultatlistorna. Avmarkera om du föredrar att vara anonym.',
		consentMarketingLabel: 'Skicka mig nyheter och inbjudningar',
		consentMarketingHelp:
			'Sällsynta utskick om kommande lopp. Du kan när som helst avregistrera dig.',
		guardianNoticeHeading: 'Målsmans samtycke krävs',
		guardianNoticeUnder13:
			'Eftersom löparen är under 13 år behöver vi en målsmans samtycke. Ange målsmans e-post ovan; vi mejlar en bekräftelselänk som måste klickas inom 7 dagar för att anmälan ska bli aktiv.',
		guardianNoticeTeen:
			'Löparen är under 18 år. Anmäl gärna med målsmans kontaktuppgifter så vi kan nå er om något händer på loppet.',
		guardianConsentLabel: 'Jag är förälder/målsman och samtycker till att barnet anmäls.',
		submit: 'Skicka anmälan',
		submitting: 'Skickar…',
		successHeading: 'Tack — vi har tagit emot din anmälan!',
		successBody: 'Vi hör av oss med mer information inför loppet.',
		successHeadingGuardian: 'Nästan klart — kolla din e-post',
		successBodyGuardian:
			'Vi har skickat en bekräftelselänk till målsmans e-post. Klicka på länken inom 7 dagar för att slutföra anmälan.',
		registerAnother: 'Anmäl en till',
		noUpcomingRaces: 'Det finns inga kommande lopp att anmäla sig till just nu.',
		eventPast: 'Det här loppet har redan ägt rum och går inte att anmäla sig till.',
		errorPrefix: 'Något gick fel:',
		botProtectionNote: 'Skyddat mot automatisk registrering.'
	},
	guardianConsent: {
		verifying: 'Verifierar bekräftelselänken …',
		successHeading: 'Tack — anmälan är nu aktiv',
		successBody: 'Din målsmans-bekräftelse är registrerad. Barnet är nu anmäld till loppet.',
		errorHeading: 'Länken kunde inte verifieras',
		errorBody:
			'Länken är ogiltig, har redan använts, eller har förfallit. Om du behöver hjälp — anmäl barnet igen så skickar vi en ny länk.',
		missingToken: 'Ingen kod fanns i URL:en.'
	},
	myData: {
		heading: 'Mina uppgifter',
		intro:
			'Här kan du se, exportera och hantera alla uppgifter vi har om dig och löpare du anmält.',
		emailLabel: 'E-post',
		emailPlaceholder: 'din@email.se',
		emailHelp: 'Vi mejlar en länk som loggar in dig på den här sidan. Inget lösenord behövs.',
		requestSubmit: 'Skicka inloggningslänk',
		requestSubmitting: 'Skickar …',
		sentHeading: 'Kontrollera din e-post',
		sentBody:
			'Om den e-postadressen finns hos oss har vi skickat en länk dit. Klicka på den inom 30 minuter för att logga in.',
		verifying: 'Loggar in …',
		errorHeading: 'Länken kunde inte verifieras',
		errorBody: 'Länken är ogiltig, har redan använts, eller har förfallit. Begär en ny länk.',
		tryAgain: 'Begär en ny länk',
		logout: 'Logga ut',
		youHeading: 'Du',
		youEmail: 'E-post',
		youCreatedAt: 'Konto skapat',
		youLocale: 'Språk',
		youMarketingConsent: 'Nyhetsbrev',
		consentGranted: 'Ja',
		consentNot: 'Nej',
		runnersHeading: (n: number) =>
			n === 1 ? '1 löpare under ditt konto' : `${n} löpare under ditt konto`,
		runnersEmpty: 'Inga löpare ännu — anmäl dig själv eller en familjemedlem.',
		runnersYears: 'år',
		runnersPublicResults: 'Publika resultat',
		runnersRegistrations: (n: number) => (n === 1 ? '1 anmälan' : `${n} anmälningar`),
		runnersUnder13Note: (name: string) =>
			`Eftersom ${name} är under 13 år visas hen aldrig publikt — inställningen ovan får effekt först när hen fyller 13.`,
		retentionIndefinite:
			'Lagring: löparhistoriken sparas tills vidare (eftersom publika resultat är tillåtna).',
		retentionAnonymizesAt: (date: string) =>
			`Lagring: namn och födelsedatum anonymiseras automatiskt ${date} (36 månader efter senaste loppet). Klockan börjar om varje gång löparen deltar i ett nytt lopp.`,
		retentionNoTimer:
			'Lagring: ingen tidsräknare igång — den startar efter ditt första genomförda lopp.',
		inactivityNote: (date: string) =>
			`Ditt konto raderas automatiskt ${date} om du inte är aktiv (loggar in, anmäler dig till ett lopp eller springer ett lopp) före dess. Klockan börjar om vid varje aktivitet.`,
		exportHeading: 'Exportera dina uppgifter',
		exportBody:
			'Ladda ner allt vi har om dig som en JSON-fil — konto, löpare, anmälningar och samtycken.',
		exportSubmit: 'Ladda ner JSON',
		exportSubmitting: 'Förbereder …',
		changeEmail: 'Byt e-post',
		changeEmailHeading: 'Byt e-postadress',
		changeEmailBody:
			'Ange din nya e-postadress. Vi skickar en bekräftelselänk dit — adressen byts först när du klickat på länken.',
		changeEmailLabel: 'Ny e-postadress',
		changeEmailSubmit: 'Skicka bekräftelselänk',
		changeEmailSent:
			'Vi har skickat en bekräftelselänk till den nya adressen. Klicka på länken inom 24 timmar för att slutföra bytet.',
		editRunner: 'Redigera löpare',
		editRunnerHeading: 'Redigera löpare',
		editRunnerName: 'Namn',
		editRunnerGender: 'Kön',
		editRunnerDob: 'Födelsedatum',
		editRunnerDobLocked:
			'Födelsedatum kan inte ändras efter att löparen har deltagit i ett lopp (åldersklassen är fastlåst).',
		confirmEmailVerifying: 'Bekräftar adressbyte …',
		confirmEmailSuccessHeading: 'E-postadressen är ändrad',
		confirmEmailSuccessBody:
			'Din nya e-postadress används nu för inloggning, bekräftelser och anmälningar:',
		confirmEmailErrorHeading: 'Länken kunde inte verifieras',
		confirmEmailErrorBody:
			'Länken är ogiltig, har redan använts, eller har förfallit. Begär ett nytt byte från Mina uppgifter.',
		confirmEmailMissingToken: 'Ingen kod fanns i URL:en.',
		confirmEmailBack: 'Till Mina uppgifter',
		save: 'Spara',
		cancel: 'Avbryt',
		close: 'Stäng',
		eraseRunner: 'Radera löpare',
		eraseRunnerHeading: 'Radera löpare?',
		eraseRunnerBody: (name: string) =>
			`Du är på väg att radera ${name}. Detta kan ångras inom 30 dagar.`,
		eraseAccountHeading: 'Radera hela ditt konto?',
		eraseAccountBody:
			'Du är på väg att radera ditt konto och alla löpare under det. Detta kan ångras inom 30 dagar.',
		eraseBulletGrace: 'Uppgifterna döljs direkt och raderas slutgiltigt efter 30 dagar.',
		eraseBulletRestoreLink: 'Vi mejlar dig en återställningslänk som gäller hela ångerperioden.',
		eraseBulletHistory:
			'Tidigare avslutade lopp behålls i resultatlistorna men anonymiseras (namn → "Anonym").',
		eraseConfirmLabel: 'Skriv RADERA för att bekräfta',
		eraseConfirmMismatch: 'Bekräftelsen matchar inte. Skriv RADERA (eller DELETE).',
		eraseSubmit: 'Radera',
		deleteAccountHeading: 'Radera kontot',
		deleteAccountBody:
			'Raderar hela ditt konto + alla löpare under det. Du kan ångra under 30 dagar via mejlet vi skickar dig.',
		deleteAccountSubmit: 'Radera mitt konto',
		deleteAccountDoneHeading: 'Kontot är schemalagt för radering',
		deleteAccountDoneBody:
			'Vi har skickat dig ett mejl med en återställningslänk. Klicka på den inom 30 dagar om du ändrar dig — annars raderas allt automatiskt.',
		restoreVerifying: 'Återställer dina uppgifter …',
		restoreSuccessHeading: 'Dina uppgifter är återställda',
		restoreSuccessBody:
			'Raderingen är avbruten och allt är tillbaka som vanligt. Du loggas nu in på Mina uppgifter.',
		restoreContinue: 'Till Mina uppgifter',
		restoreErrorHeading: 'Länken kunde inte verifieras',
		restoreErrorBody:
			'Länken är ogiltig, har redan använts, eller har förfallit (kanske för att 30-dagarsfönstret tagit slut).',
		restoreMissingToken: 'Ingen kod fanns i URL:en.',
		restoreBack: 'Till Mina uppgifter',
		pendingDeletionHeading: (date: string) => `Ditt konto är schemalagt för radering den ${date}`,
		pendingDeletionBody:
			'Uppgifterna är dolda från publika listor men kan återställas till och med raderingsdatumet. Klicka nedan om du ångrar dig.',
		pendingDeletionCancel: 'Återställ kontot',
		activityHeading: 'Din aktivitet',
		activityBody:
			'Här ser du vad som hänt med dina uppgifter — när du loggat in, ändrat samtycken, eller när vi mejlat dig.',
		activityActorUser: 'Du',
		activityActorAdmin: 'Administratör',
		activityActorSystem: 'System',
		activityUpcoming: 'Planerat',
		activityUpcomingDeletion: 'Kontot raderas slutgiltigt',
		activityActions: {
			'dsr.session.started': 'Loggade in på Mina uppgifter',
			'consent.marketing.update': 'Uppdaterade nyhetsbrev-samtycke',
			'consent.publicResults.update': 'Uppdaterade samtycke för publika resultat',
			'locale.update': 'Ändrade språk',
			'runner.update': 'Uppdaterade löparuppgifter',
			'email.changed': 'Bytte e-postadress',
			'email.sent': 'Vi skickade ett mejl till dig',
			'email.send.failed': 'Försökte mejla dig — leveransen misslyckades',
			'runner.erased': 'Schemalade löpare för radering',
			'account.erased': 'Schemalade kontot för radering',
			'account.restored': 'Återställde kontot från radering',
			'admin.registration.created': 'Administratör skapade en anmälan',
			'admin.registration.updated': 'Administratör uppdaterade en anmälan',
			'admin.registration.deleted': 'Administratör tog bort en anmälan',
			'retention.runner.anonymized':
				'Vi anonymiserade dina löparuppgifter enligt 36-månaderspolicyn',
			'retention.account.inactivity':
				'Vi schemalade ditt konto för radering efter 36 månaders inaktivitet'
		}
	},
	admin: {
		title: 'Admin',
		login: {
			heading: 'Logga in',
			subheading: 'Logga in för att hantera lopp, löpare och resultat.',
			emailLabel: 'E-post',
			passwordLabel: 'Lösenord',
			submit: 'Logga in',
			submitting: 'Loggar in…',
			verifyHeading: 'Tvåfaktorskod',
			verifySubheading: 'Ange den 6-siffriga koden från din autentiseringsapp.',
			enrollHeading: 'Aktivera tvåfaktorsautentisering',
			enrollSubheading:
				'Skanna QR-koden eller lägg till nyckeln manuellt i en autentiseringsapp och ange den 6-siffriga koden.',
			enrollInstructions:
				'Skanna länken nedan med en autentiseringsapp (t.ex. 1Password, Authy, Google Authenticator).',
			enrollSecretLabel: 'Manuell nyckel:',
			codeLabel: 'Engångskod',
			codeSubmit: 'Verifiera'
		},
		accept: {
			heading: 'Acceptera inbjudan',
			iterationNotice:
				'Inbjudningsflöde är inte aktivt än. Be administratören att skapa kontot via just create-admin.',
			backToLogin: 'Till inloggning'
		},
		nav: {
			dashboard: 'Översikt',
			runners: 'Löpare',
			events: 'Lopp',
			results: 'Resultat',
			users: 'Användare',
			signedInAs: (email: string) => `Inloggad som ${email}`,
			logout: 'Logga ut',
			viewPublicSite: 'Visa publik sida'
		},
		dashboard: {
			heading: 'Översikt',
			subheading: 'Snabb överblick av allt som finns i systemet.',
			runners: 'Löpare',
			events: 'Lopp',
			races: 'Klasser',
			results: 'Resultat',
			users: 'Adminanvändare',
			pendingInvites: (n: number) => (n === 1 ? '1 inbjudan väntar' : `${n} inbjudningar väntar`)
		},
		common: {
			cancel: 'Avbryt',
			save: 'Spara',
			saving: 'Sparar…',
			create: 'Skapa',
			creating: 'Skapar…',
			delete: 'Ta bort',
			deleting: 'Tar bort…',
			edit: 'Redigera',
			confirmDelete: 'Är du säker? Detta kan inte ångras.',
			search: 'Sök…',
			id: 'ID',
			actions: 'Åtgärder',
			noneYet: 'Inga ännu.',
			loading: 'Laddar…',
			error: (msg: string) => `Fel: ${msg}`
		},
		runners: {
			heading: 'Löpare',
			add: 'Lägg till löpare',
			edit: 'Redigera löpare',
			nameLabel: 'Namn',
			genderLabel: 'Kön',
			birthYearLabel: 'Födelseår',
			columnName: 'Namn',
			columnGender: 'Kön',
			columnBirthYear: 'Födelseår',
			registerForRace: 'Anmäl till klass',
			registerHeading: (name: string) => `Anmäl ${name}`,
			raceSelectLabel: 'Klass',
			noRacesAvailable: 'Den här löparen är redan anmäld till alla klasser.'
		},
		results: {
			heading: 'Resultat',
			subheading: 'Välj år och klass för att registrera tider.',
			yearLabel: 'År',
			raceLabel: 'Klass',
			pickPrompt: 'Välj år och klass ovan för att se anmälda löpare.'
		},
		events: {
			heading: 'Lopp',
			add: 'Skapa nytt år',
			edit: 'Redigera lopp',
			nameLabel: 'Namn',
			yearLabel: 'År',
			dateLabel: 'Datum',
			locationLabel: 'Plats',
			racesHeading: 'Klasser i detta lopp',
			addRace: 'Lägg till klass',
			editRace: 'Redigera klass',
			raceNameLabel: 'Klassnamn',
			distanceLabel: 'Distans (meter)',
			disciplineLabel: 'Disciplin',
			columnName: 'Namn',
			columnYear: 'År',
			columnDate: 'Datum',
			columnLocation: 'Plats',
			columnRaces: 'Antal klasser'
		},
		races: {
			pageBack: 'Tillbaka till loppet',
			pageHeading: 'Anmälningar och resultat',
			registerHeading: 'Anmäl löpare',
			registerSubmit: 'Anmäl',
			runnerSelectLabel: 'Löpare',
			bibLabel: 'Startnummer',
			noRunnersAvailable: 'Alla löpare är redan anmälda till denna klass.',
			registeredHeading: (n: number) => (n === 1 ? '1 anmäld löpare' : `${n} anmälda löpare`),
			columnRunner: 'Löpare',
			columnBib: 'Startnr',
			columnCategory: 'Kategori',
			columnStatus: 'Status',
			columnTime: 'Tid (mm:ss eller h:mm:ss)',
			status: {
				pending: 'Väntar',
				finished: 'Klar',
				dnf: 'DNF',
				dns: 'DNS'
			},
			saveChanges: 'Spara ändringar',
			saveChangesWithCount: (n: number) => (n === 1 ? 'Spara 1 ändring' : `Spara ${n} ändringar`),
			nothingToSave: 'Inga ändringar att spara.',
			saveSuccess: 'Sparat.',
			invalidTime: 'Ogiltigt tidsformat. Använd mm:ss eller h:mm:ss.',
			mustHaveFinishTime: 'Status "Klar" kräver en tid.',
			confirmRemoveRegistration:
				'Ta bort denna anmälan? Eventuell tid och anteckningar försvinner.',
			emptyState: 'Inga löpare är anmälda än. Använd formuläret ovan för att anmäla.'
		},
		users: {
			heading: 'Adminanvändare',
			iterationNotice:
				'Inbjudningsflöde och hantering av flera adminanvändare kommer i en senare iteration. Använd just create-admin från terminalen för att skapa fler administratörer.',
			currentSession: (email: string) => `Inloggad som ${email}.`
		}
	}
};

type Catalog = typeof sv;

const en: Catalog = {
	nav: {
		brand: 'Ingmarsöloppet',
		home: 'Home',
		runners: 'Runners',
		register: 'Sign up',
		myData: 'My data',
		language: 'Language',
		theme: 'Theme',
		themeLight: 'Light',
		themeDark: 'Dark',
		themeSystem: 'System'
	},
	landing: {
		description: 'A platform for viewing runner race results across yearly events.',
		heroTagline:
			"Running joy in Stockholm's archipelago — results, stats and history from every edition.",
		ctaEvents: 'View events',
		ctaRunners: 'Browse runners',
		ctaRegister: 'Sign up for a race',
		mockTitle: 'Mock data mode',
		mockDescription:
			'The pages below render hand-rolled fixtures. The backend will be wired up after the UI shape is locked in.',
		eventsHeading: 'Events',
		runnersHeading: 'Runners',
		seeAll: 'See all →',
		apiStatus: 'API status (smoke test)',
		loading: 'Loading…',
		error: (msg: string) => `Error: ${msg}`
	},
	runnersList: {
		heading: 'Runners',
		description: 'Click a runner to see their results.',
		searchPlaceholder: 'Search by name…',
		searchLabel: 'Search runners',
		resultCount: (n: number) => (n === 1 ? '1 runner' : `${n} runners`),
		noMatch: (q: string) => `No runners match "${q}".`
	},
	runner: {
		loading: 'Loading…',
		notFound: 'Runner not found.',
		backToRunners: 'Back to runners',
		bornIn: (year: number) => `born ${year}`,
		paceOverTime: 'Pace over time (run only)',
		distanceDistribution: 'Distance distribution',
		resultsHeading: 'Results'
	},
	result: {
		loading: 'Loading…',
		notFound: 'Result not found.',
		backToRunners: 'Back to runners',
		distance: 'Distance',
		finish: 'Finish',
		pace: 'Pace',
		placement: 'Placement',
		placementCategory: (place: number, group: string) => `#${place} in ${group}`,
		bib: 'Bib',
		category: 'Category',
		splitsHeading: 'Splits',
		splitsBarTitle: 'Pace per km',
		fastestKmLabel: 'fastest',
		conditionsHeading: 'Conditions',
		notesHeading: 'Notes'
	},
	illustrations: {
		decorative: 'Decorative'
	},
	resultsTable: {
		filter: 'Filter:',
		columnDate: 'Date',
		columnEventRace: 'Event / Race',
		columnDistance: 'Distance',
		columnTime: 'Time',
		columnPace: 'Pace',
		columnPlace: 'Place',
		columnBib: 'Bib',
		noMatch: 'No results match this filter.'
	},
	event: {
		loading: 'Loading…',
		notFound: 'Event not found.',
		backHome: 'Back to home',
		racesHeading: 'Races',
		participants: (count: number) => (count === 1 ? '1 participant' : `${count} participants`),
		noResults: 'No results yet.'
	},
	race: {
		kids: "Kids' race"
	},
	category: {
		gender: { M: 'M', F: 'F', X: 'X' } as Record<Gender, string>
	},
	leaderboard: {
		columnRank: 'Rank',
		columnRunner: 'Runner',
		columnTime: 'Time',
		columnPace: 'Pace',
		columnCategory: 'Category',
		columnBib: 'Bib'
	},
	discipline: {
		all: 'all',
		run: 'run',
		walk: 'walk',
		kids: 'kids'
	},
	summary: {
		races: 'Races',
		acrossEvents: (count: number) => `across ${count} event${count === 1 ? '' : 's'}`,
		totalDistance: 'Total distance',
		personalBests: 'Personal bests (run)'
	},
	splits: {
		none: 'No splits recorded.',
		km: 'Km',
		split: 'Split',
		cumulative: 'Cumulative',
		fastest: (km: number, time: string) => `Fastest km: km ${km} in ${time}.`
	},
	chart: {
		paceHelp: 'Lower is faster. Hover points for race details.',
		noRunsYet: 'No run results yet.',
		noResultsYet: 'No results yet.'
	},
	register: {
		heading: 'Sign up for a race',
		subheading: 'Fill in your details and pick which race you want to run.',
		nameLabel: 'Name',
		namePlaceholder: 'e.g. Anna Andersson',
		emailLabel: 'Email',
		emailPlaceholder: 'you@example.com',
		emailHelp: 'Used for confirmation, race-day information, and to manage your data.',
		dobLabel: 'Date of birth',
		genderLabel: 'Gender',
		raceLabel: 'Race',
		racePlaceholder: 'Pick an upcoming race',
		consentPublicResultsLabel: 'Show my result publicly',
		consentPublicResultsHelp:
			'Your name, category and time appear in the public results listings. Uncheck if you prefer to stay anonymous.',
		consentMarketingLabel: 'Send me race news and invitations',
		consentMarketingHelp:
			'Occasional updates about upcoming races. You can unsubscribe at any time.',
		guardianNoticeHeading: 'Guardian consent required',
		guardianNoticeUnder13:
			"Because the runner is under 13, we need a parent or guardian's consent. Enter the guardian's email above; we'll send a confirmation link that must be clicked within 7 days for the registration to take effect.",
		guardianNoticeTeen:
			'The runner is under 18. Please use a parent or guardian as the contact so we can reach you if anything happens on race day.',
		guardianConsentLabel: 'I am the parent/guardian and consent to the registration of this child.',
		submit: 'Submit registration',
		submitting: 'Submitting…',
		successHeading: "Thanks — we've received your registration!",
		successBody: "We'll be in touch with more information ahead of the race.",
		successHeadingGuardian: 'Almost done — check your email',
		successBodyGuardian:
			"We've sent a confirmation link to the guardian's email. Click it within 7 days to complete the registration.",
		registerAnother: 'Sign someone else up',
		noUpcomingRaces: 'There are no upcoming races to sign up for right now.',
		eventPast: 'This race has already taken place and is no longer open for sign-ups.',
		errorPrefix: 'Something went wrong:',
		botProtectionNote: 'Protected against automated sign-ups.'
	},
	guardianConsent: {
		verifying: 'Verifying confirmation link …',
		successHeading: 'Thanks — the registration is now active',
		successBody:
			'Your guardian consent has been recorded. The child is now registered for the race.',
		errorHeading: 'The link could not be verified',
		errorBody:
			"The link is invalid, already used, or expired. If you need help, register the child again — we'll send a new link.",
		missingToken: 'No token was supplied in the URL.'
	},
	myData: {
		heading: 'My data',
		intro:
			'See, export and manage everything we have on file about you and the runners you have signed up.',
		emailLabel: 'Email',
		emailPlaceholder: 'you@example.com',
		emailHelp: "We'll email you a link that signs you in to this page. No password needed.",
		requestSubmit: 'Send sign-in link',
		requestSubmitting: 'Sending …',
		sentHeading: 'Check your email',
		sentBody:
			"If we have that email on file, we've sent a link there. Click it within 30 minutes to sign in.",
		verifying: 'Signing you in …',
		errorHeading: 'The link could not be verified',
		errorBody: 'The link is invalid, already used, or expired. Request a new one.',
		tryAgain: 'Request a new link',
		logout: 'Sign out',
		youHeading: 'You',
		youEmail: 'Email',
		youCreatedAt: 'Account created',
		youLocale: 'Language',
		youMarketingConsent: 'Newsletter',
		consentGranted: 'Yes',
		consentNot: 'No',
		runnersHeading: (n: number) =>
			n === 1 ? '1 runner under your account' : `${n} runners under your account`,
		runnersEmpty: 'No runners yet — sign yourself or a family member up.',
		runnersYears: 'yrs',
		runnersPublicResults: 'Public results',
		runnersRegistrations: (n: number) => (n === 1 ? '1 registration' : `${n} registrations`),
		runnersUnder13Note: (name: string) =>
			`While ${name} is under 13, they are never shown publicly — the setting above will only take effect when they turn 13.`,
		retentionIndefinite: 'Retention: race history kept indefinitely (you allow public results).',
		retentionAnonymizesAt: (date: string) =>
			`Retention: name and date of birth are auto-anonymized on ${date} (36 months after the last race). The clock resets each time the runner participates in a new race.`,
		retentionNoTimer: 'Retention: no timer running yet — it starts after your first finished race.',
		inactivityNote: (date: string) =>
			`Your account will be deleted automatically on ${date} unless you're active (log in, register for a race, or finish a race) before then. The clock resets on every activity.`,
		exportHeading: 'Export your data',
		exportBody:
			'Download everything we have on you as a JSON file — account, runners, registrations, consents.',
		exportSubmit: 'Download JSON',
		exportSubmitting: 'Preparing …',
		changeEmail: 'Change email',
		changeEmailHeading: 'Change email address',
		changeEmailBody:
			"Enter your new email address. We'll send a confirmation link there — the change takes effect when you click it.",
		changeEmailLabel: 'New email address',
		changeEmailSubmit: 'Send confirmation link',
		changeEmailSent:
			"We've sent a confirmation link to the new address. Click it within 24 hours to complete the change.",
		editRunner: 'Edit runner',
		editRunnerHeading: 'Edit runner',
		editRunnerName: 'Name',
		editRunnerGender: 'Gender',
		editRunnerDob: 'Date of birth',
		editRunnerDobLocked:
			'Date of birth cannot be changed once this runner has participated in a race (age category is locked).',
		confirmEmailVerifying: 'Confirming address change …',
		confirmEmailSuccessHeading: 'Email address changed',
		confirmEmailSuccessBody:
			'Your new email address is now used for sign-in, confirmations and registrations:',
		confirmEmailErrorHeading: 'The link could not be verified',
		confirmEmailErrorBody:
			'The link is invalid, already used, or expired. Request a new change from My data.',
		confirmEmailMissingToken: 'No token was supplied in the URL.',
		confirmEmailBack: 'Back to My data',
		save: 'Save',
		cancel: 'Cancel',
		close: 'Close',
		eraseRunner: 'Erase runner',
		eraseRunnerHeading: 'Erase runner?',
		eraseRunnerBody: (name: string) =>
			`You are about to erase ${name}. This can be undone within 30 days.`,
		eraseAccountHeading: 'Erase your entire account?',
		eraseAccountBody:
			'You are about to erase your account and every runner under it. This can be undone within 30 days.',
		eraseBulletGrace: 'Your data is hidden immediately and permanently deleted after 30 days.',
		eraseBulletRestoreLink: "We'll email you a restore link valid for the full undo window.",
		eraseBulletHistory:
			'Past finished races stay in the results lists but are anonymized (name → "Anonym").',
		eraseConfirmLabel: 'Type DELETE to confirm',
		eraseConfirmMismatch: 'Confirmation does not match. Type DELETE (or RADERA).',
		eraseSubmit: 'Erase',
		deleteAccountHeading: 'Delete account',
		deleteAccountBody:
			'Erases your entire account + every runner under it. You can undo within 30 days via the email we send you.',
		deleteAccountSubmit: 'Delete my account',
		deleteAccountDoneHeading: 'Account scheduled for deletion',
		deleteAccountDoneBody:
			"We've emailed you a restore link. Click it within 30 days if you change your mind — otherwise everything is deleted automatically.",
		restoreVerifying: 'Restoring your data …',
		restoreSuccessHeading: 'Your data is restored',
		restoreSuccessBody:
			"Deletion was undone and everything is back to normal. You're now signed in to My data.",
		restoreContinue: 'Go to My data',
		restoreErrorHeading: 'The link could not be verified',
		restoreErrorBody:
			'The link is invalid, already used, or expired (perhaps the 30-day window closed).',
		restoreMissingToken: 'No token was supplied in the URL.',
		restoreBack: 'Back to My data',
		pendingDeletionHeading: (date: string) => `Your account is scheduled for deletion on ${date}`,
		pendingDeletionBody:
			'Your data is hidden from public listings but can still be restored until the deletion date. Click below if you change your mind.',
		pendingDeletionCancel: 'Restore my account',
		activityHeading: 'Your activity',
		activityBody:
			"This is everything we've done with your data — sign-ins, consent changes, emails we've sent you.",
		activityActorUser: 'You',
		activityActorAdmin: 'Admin',
		activityActorSystem: 'System',
		activityUpcoming: 'Scheduled',
		activityUpcomingDeletion: 'Account permanently deleted',
		activityActions: {
			'dsr.session.started': 'Signed in to My data',
			'consent.marketing.update': 'Updated newsletter consent',
			'consent.publicResults.update': 'Updated public-results consent',
			'locale.update': 'Changed language',
			'runner.update': 'Updated runner details',
			'email.changed': 'Changed email address',
			'email.sent': 'We sent you an email',
			'email.send.failed': 'Tried to email you — delivery failed',
			'runner.erased': 'Scheduled runner for erasure',
			'account.erased': 'Scheduled account for erasure',
			'account.restored': 'Restored account from pending erasure',
			'admin.registration.created': 'An admin created a registration',
			'admin.registration.updated': 'An admin updated a registration',
			'admin.registration.deleted': 'An admin deleted a registration',
			'retention.runner.anonymized':
				'We anonymized your runner data per the 36-month retention policy',
			'retention.account.inactivity':
				'We scheduled your account for deletion after 36 months of inactivity'
		}
	},
	admin: {
		title: 'Admin',
		login: {
			heading: 'Sign in',
			subheading: 'Sign in to manage events, runners and results.',
			emailLabel: 'Email',
			passwordLabel: 'Password',
			submit: 'Sign in',
			submitting: 'Signing in…',
			verifyHeading: 'Two-factor code',
			verifySubheading: 'Enter the 6-digit code from your authenticator app.',
			enrollHeading: 'Enable two-factor authentication',
			enrollSubheading:
				'Scan the QR or add the key manually in an authenticator app, then enter the 6-digit code.',
			enrollInstructions:
				'Add the link below to an authenticator app (e.g. 1Password, Authy, Google Authenticator).',
			enrollSecretLabel: 'Manual key:',
			codeLabel: 'One-time code',
			codeSubmit: 'Verify'
		},
		accept: {
			heading: 'Accept invitation',
			iterationNotice:
				'Invitation flow is not active yet. Ask the admin to create the account using just create-admin.',
			backToLogin: 'Back to sign-in'
		},
		nav: {
			dashboard: 'Overview',
			runners: 'Runners',
			events: 'Events',
			results: 'Results',
			users: 'Users',
			signedInAs: (email: string) => `Signed in as ${email}`,
			logout: 'Sign out',
			viewPublicSite: 'View public site'
		},
		dashboard: {
			heading: 'Overview',
			subheading: 'Quick snapshot of everything in the system.',
			runners: 'Runners',
			events: 'Events',
			races: 'Races',
			results: 'Results',
			users: 'Admin users',
			pendingInvites: (n: number) => (n === 1 ? '1 invite pending' : `${n} invites pending`)
		},
		common: {
			cancel: 'Cancel',
			save: 'Save',
			saving: 'Saving…',
			create: 'Create',
			creating: 'Creating…',
			delete: 'Delete',
			deleting: 'Deleting…',
			edit: 'Edit',
			confirmDelete: 'Are you sure? This cannot be undone.',
			search: 'Search…',
			id: 'ID',
			actions: 'Actions',
			noneYet: 'Nothing here yet.',
			loading: 'Loading…',
			error: (msg: string) => `Error: ${msg}`
		},
		runners: {
			heading: 'Runners',
			add: 'Add runner',
			edit: 'Edit runner',
			nameLabel: 'Name',
			genderLabel: 'Gender',
			birthYearLabel: 'Birth year',
			columnName: 'Name',
			columnGender: 'Gender',
			columnBirthYear: 'Birth year',
			registerForRace: 'Register for a race',
			registerHeading: (name: string) => `Register ${name}`,
			raceSelectLabel: 'Race',
			noRacesAvailable: 'This runner is already registered for every race.'
		},
		results: {
			heading: 'Results',
			subheading: 'Pick a year and race to enter finish times.',
			yearLabel: 'Year',
			raceLabel: 'Race',
			pickPrompt: 'Pick a year and race above to see the registered runners.'
		},
		events: {
			heading: 'Events',
			add: 'Add new year',
			edit: 'Edit event',
			nameLabel: 'Name',
			yearLabel: 'Year',
			dateLabel: 'Date',
			locationLabel: 'Location',
			racesHeading: 'Races in this event',
			addRace: 'Add race',
			editRace: 'Edit race',
			raceNameLabel: 'Race name',
			distanceLabel: 'Distance (meters)',
			disciplineLabel: 'Discipline',
			columnName: 'Name',
			columnYear: 'Year',
			columnDate: 'Date',
			columnLocation: 'Location',
			columnRaces: 'Races'
		},
		races: {
			pageBack: 'Back to event',
			pageHeading: 'Registrations & results',
			registerHeading: 'Register runner',
			registerSubmit: 'Register',
			runnerSelectLabel: 'Runner',
			bibLabel: 'Bib',
			noRunnersAvailable: 'Every runner is already registered for this race.',
			registeredHeading: (n: number) =>
				n === 1 ? '1 registered runner' : `${n} registered runners`,
			columnRunner: 'Runner',
			columnBib: 'Bib',
			columnCategory: 'Category',
			columnStatus: 'Status',
			columnTime: 'Time (mm:ss or h:mm:ss)',
			status: {
				pending: 'Pending',
				finished: 'Finished',
				dnf: 'DNF',
				dns: 'DNS'
			},
			saveChanges: 'Save changes',
			saveChangesWithCount: (n: number) => (n === 1 ? 'Save 1 change' : `Save ${n} changes`),
			nothingToSave: 'No changes to save.',
			saveSuccess: 'Saved.',
			invalidTime: 'Invalid time format. Use mm:ss or h:mm:ss.',
			mustHaveFinishTime: 'Finished status requires a finish time.',
			confirmRemoveRegistration:
				'Remove this registration? Any recorded time and notes will be discarded.',
			emptyState: 'No runners registered yet. Use the form above to register.'
		},
		users: {
			heading: 'Admin users',
			iterationNotice:
				'Invitation flow and multi-admin management is a later iteration. Use just create-admin from the terminal to add more administrators.',
			currentSession: (email: string) => `Signed in as ${email}.`
		}
	}
};

export const messages: Record<Locale, Catalog> = { sv, en };

export type Messages = Catalog;
