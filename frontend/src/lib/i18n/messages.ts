export type Locale = 'sv' | 'en';

export const locales: ReadonlyArray<{ code: Locale; name: string }> = [
	{ code: 'sv', name: 'Svenska' },
	{ code: 'en', name: 'English' }
];

const sv = {
	nav: {
		brand: 'the-run',
		home: 'Hem',
		runners: 'Löpare',
		language: 'Språk'
	},
	footer: 'the-run · mockdata — backend ej ansluten',
	landing: {
		description: 'En plattform för att visa löparresultat från årliga lopp.',
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
		conditionsHeading: 'Förhållanden',
		notesHeading: 'Anteckningar'
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
		racesHeading: 'Klasser',
		participants: (count: number) => (count === 1 ? '1 deltagare' : `${count} deltagare`),
		noResults: 'Inga resultat ännu.'
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
	}
};

type Catalog = typeof sv;

const en: Catalog = {
	nav: {
		brand: 'the-run',
		home: 'Home',
		runners: 'Runners',
		language: 'Language'
	},
	footer: 'the-run · mock data — backend not yet wired',
	landing: {
		description: 'A platform for viewing runner race results across yearly events.',
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
		conditionsHeading: 'Conditions',
		notesHeading: 'Notes'
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
	}
};

export const messages: Record<Locale, Catalog> = { sv, en };

export type Messages = Catalog;
