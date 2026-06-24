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
		language: 'Språk',
		theme: 'Tema',
		themeLight: 'Ljust',
		themeDark: 'Mörkt',
		themeSystem: 'System'
	},
	footer: 'Ingmarsöloppet · mockdata — backend ej ansluten',
	landing: {
		description: 'En plattform för att visa löparresultat från årliga lopp.',
		heroTagline: 'Löparglädje i Stockholms skärgård — resultat, statistik och historik från varje upplaga.',
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
		dobLabel: 'Födelsedatum',
		genderLabel: 'Kön',
		raceLabel: 'Lopp',
		racePlaceholder: 'Välj ett kommande lopp',
		submit: 'Skicka anmälan',
		submitting: 'Skickar…',
		successHeading: 'Tack — vi har tagit emot din anmälan!',
		successBody: 'Vi hör av oss med mer information inför loppet.',
		registerAnother: 'Anmäl en till',
		noUpcomingRaces: 'Det finns inga kommande lopp att anmäla sig till just nu.',
		eventPast: 'Det här loppet har redan ägt rum och går inte att anmäla sig till.',
		errorPrefix: 'Något gick fel:',
		botProtectionNote: 'Skyddat mot automatisk registrering.'
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
			mockHint: (email: string, password: string) =>
				`Mockläge: använd ${email} / ${password} för att logga in.`
		},
		accept: {
			heading: 'Acceptera inbjudan',
			subheading: 'Sätt ett lösenord för att slutföra din registrering.',
			emailLabel: 'E-post',
			passwordLabel: 'Nytt lösenord',
			passwordHint: 'Minst 6 tecken.',
			submit: 'Slutför',
			submitting: 'Sparar…',
			missingToken: 'Saknar inbjudningstoken.',
			alreadyAccepted: 'Inbjudan är redan använd.',
			success: 'Klart — du är inloggad.'
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
			saveChangesWithCount: (n: number) =>
				n === 1 ? 'Spara 1 ändring' : `Spara ${n} ändringar`,
			nothingToSave: 'Inga ändringar att spara.',
			saveSuccess: 'Sparat.',
			invalidTime: 'Ogiltigt tidsformat. Använd mm:ss eller h:mm:ss.',
			mustHaveFinishTime: 'Status "Klar" kräver en tid.',
			confirmRemoveRegistration: 'Ta bort denna anmälan? Eventuell tid och anteckningar försvinner.',
			emptyState: 'Inga löpare är anmälda än. Använd formuläret ovan för att anmäla.'
		},
		users: {
			heading: 'Adminanvändare',
			invite: 'Bjud in användare',
			inviteHeading: 'Bjud in ny användare',
			inviteEmailLabel: 'E-post',
			inviteSubmit: 'Skicka inbjudan',
			inviteCreated: 'Inbjudan skapad. Skicka denna länk till mottagaren:',
			pendingHeading: 'Väntande inbjudningar',
			usersHeading: 'Aktiva användare',
			columnEmail: 'E-post',
			columnCreated: 'Skapad',
			columnInvitedBy: 'Inbjuden av',
			pendingBadge: 'Väntar',
			revoke: 'Återkalla',
			youBadge: 'Du'
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
		language: 'Language',
		theme: 'Theme',
		themeLight: 'Light',
		themeDark: 'Dark',
		themeSystem: 'System'
	},
	footer: 'Ingmarsöloppet · mock data — backend not yet wired',
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
		dobLabel: 'Date of birth',
		genderLabel: 'Gender',
		raceLabel: 'Race',
		racePlaceholder: 'Pick an upcoming race',
		submit: 'Submit registration',
		submitting: 'Submitting…',
		successHeading: "Thanks — we've received your registration!",
		successBody: "We'll be in touch with more information ahead of the race.",
		registerAnother: 'Sign someone else up',
		noUpcomingRaces: 'There are no upcoming races to sign up for right now.',
		eventPast: 'This race has already taken place and is no longer open for sign-ups.',
		errorPrefix: 'Something went wrong:',
		botProtectionNote: 'Protected against automated sign-ups.'
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
			mockHint: (email: string, password: string) =>
				`Mock mode: log in with ${email} / ${password}.`
		},
		accept: {
			heading: 'Accept invitation',
			subheading: 'Set a password to finish creating your account.',
			emailLabel: 'Email',
			passwordLabel: 'New password',
			passwordHint: 'At least 6 characters.',
			submit: 'Finish',
			submitting: 'Saving…',
			missingToken: 'Missing invitation token.',
			alreadyAccepted: 'This invitation has already been used.',
			success: "You're signed in."
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
			registeredHeading: (n: number) => (n === 1 ? '1 registered runner' : `${n} registered runners`),
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
			saveChangesWithCount: (n: number) =>
				n === 1 ? 'Save 1 change' : `Save ${n} changes`,
			nothingToSave: 'No changes to save.',
			saveSuccess: 'Saved.',
			invalidTime: 'Invalid time format. Use mm:ss or h:mm:ss.',
			mustHaveFinishTime: 'Finished status requires a finish time.',
			confirmRemoveRegistration: 'Remove this registration? Any recorded time and notes will be discarded.',
			emptyState: 'No runners registered yet. Use the form above to register.'
		},
		users: {
			heading: 'Admin users',
			invite: 'Invite user',
			inviteHeading: 'Invite a new user',
			inviteEmailLabel: 'Email',
			inviteSubmit: 'Send invitation',
			inviteCreated: 'Invitation created. Send this link to the recipient:',
			pendingHeading: 'Pending invitations',
			usersHeading: 'Active users',
			columnEmail: 'Email',
			columnCreated: 'Created',
			columnInvitedBy: 'Invited by',
			pendingBadge: 'Pending',
			revoke: 'Revoke',
			youBadge: 'You'
		}
	}
};

export const messages: Record<Locale, Catalog> = { sv, en };

export type Messages = Catalog;
