import type { Race, RaceEvent, Result, Runner } from '$lib/types';

export const runners: Runner[] = [
	{ id: 'r-birger', name: 'Birger Rydbäck', gender: 'M', birthYear: 1988 },
	{ id: 'r-anna', name: 'Anna Lindqvist', gender: 'F', birthYear: 1991 },
	{ id: 'r-johan', name: 'Johan Eriksson', gender: 'M', birthYear: 1975 },
	{ id: 'r-mira', name: 'Mira Sundberg', gender: 'F', birthYear: 2014 },
	{ id: 'r-oskar', name: 'Oskar Holm', gender: 'M', birthYear: 1999 }
];

export const events: RaceEvent[] = [
	{
		id: 'e-2025',
		name: 'The Run 2025',
		year: 2025,
		date: '2025-06-14',
		location: 'Stockholm'
	},
	{
		id: 'e-2026',
		name: 'The Run 2026',
		year: 2026,
		date: '2026-06-13',
		location: 'Stockholm'
	}
];

export const races: Race[] = [
	{
		id: 'rc-2025-5k-run',
		eventId: 'e-2025',
		name: '5K Run',
		distanceMeters: 5000,
		discipline: 'run'
	},
	{
		id: 'rc-2025-5k-walk',
		eventId: 'e-2025',
		name: '5K Walk',
		distanceMeters: 5000,
		discipline: 'walk'
	},
	{
		id: 'rc-2025-10k-run',
		eventId: 'e-2025',
		name: '10K Run',
		distanceMeters: 10000,
		discipline: 'run'
	},
	{
		id: 'rc-2025-kids',
		eventId: 'e-2025',
		name: "Kids' Race",
		distanceMeters: 1000,
		discipline: 'kids'
	},
	{
		id: 'rc-2026-5k-run',
		eventId: 'e-2026',
		name: '5K Run',
		distanceMeters: 5000,
		discipline: 'run'
	},
	{
		id: 'rc-2026-5k-walk',
		eventId: 'e-2026',
		name: '5K Walk',
		distanceMeters: 5000,
		discipline: 'walk'
	},
	{
		id: 'rc-2026-10k-run',
		eventId: 'e-2026',
		name: '10K Run',
		distanceMeters: 10000,
		discipline: 'run'
	},
	{
		id: 'rc-2026-kids',
		eventId: 'e-2026',
		name: "Kids' Race",
		distanceMeters: 1000,
		discipline: 'kids'
	}
];

const splits5k = (total: number): { km: number; timeSeconds: number }[] => {
	const per = total / 5;
	return Array.from({ length: 5 }, (_, i) => ({
		km: i + 1,
		timeSeconds: Math.round(per + (i - 2) * 3)
	}));
};

const splits10k = (total: number): { km: number; timeSeconds: number }[] => {
	const per = total / 10;
	return Array.from({ length: 10 }, (_, i) => ({
		km: i + 1,
		timeSeconds: Math.round(per + (i - 5) * 4)
	}));
};

export const results: Result[] = [
	{
		id: 'res-1',
		raceId: 'rc-2025-5k-run',
		runnerId: 'r-birger',
		bib: '142',
		finishSeconds: 24 * 60 + 12,
		category: { gender: 'M', ageGroup: 'M30-39' },
		placementOverall: 38,
		placementCategory: 12,
		splits: splits5k(24 * 60 + 12),
		conditions: 'Sunny, 18°C, light wind',
		notes: 'Pushed hard on the last km. New 5K PB.'
	},
	{
		id: 'res-2',
		raceId: 'rc-2025-10k-run',
		runnerId: 'r-birger',
		bib: '142',
		finishSeconds: 50 * 60 + 47,
		category: { gender: 'M', ageGroup: 'M30-39' },
		placementOverall: 64,
		placementCategory: 21,
		splits: splits10k(50 * 60 + 47),
		conditions: 'Sunny, 18°C',
		notes: 'Felt good through 7K, faded a bit.'
	},
	{
		id: 'res-3',
		raceId: 'rc-2026-5k-run',
		runnerId: 'r-birger',
		bib: '88',
		finishSeconds: 23 * 60 + 41,
		category: { gender: 'M', ageGroup: 'M30-39' },
		placementOverall: 29,
		placementCategory: 9,
		splits: splits5k(23 * 60 + 41),
		conditions: 'Overcast, 14°C',
		notes: 'New 5K PB. Went out fast and held on.'
	},
	{
		id: 'res-4',
		raceId: 'rc-2026-10k-run',
		runnerId: 'r-birger',
		bib: '88',
		finishSeconds: 49 * 60 + 8,
		category: { gender: 'M', ageGroup: 'M30-39' },
		placementOverall: 51,
		placementCategory: 16,
		splits: splits10k(49 * 60 + 8),
		conditions: 'Overcast, 15°C',
		notes: ''
	},
	{
		id: 'res-5',
		raceId: 'rc-2025-5k-walk',
		runnerId: 'r-anna',
		bib: '301',
		finishSeconds: 38 * 60 + 22,
		category: { gender: 'F', ageGroup: 'F30-39' },
		placementOverall: 4,
		placementCategory: 2,
		splits: splits5k(38 * 60 + 22),
		conditions: 'Sunny, 18°C',
		notes: 'Great pacing.'
	},
	{
		id: 'res-6',
		raceId: 'rc-2026-5k-walk',
		runnerId: 'r-anna',
		bib: '301',
		finishSeconds: 37 * 60 + 50,
		category: { gender: 'F', ageGroup: 'F30-39' },
		placementOverall: 3,
		placementCategory: 1,
		splits: splits5k(37 * 60 + 50),
		conditions: 'Overcast, 15°C',
		notes: 'First place in category!'
	},
	{
		id: 'res-7',
		raceId: 'rc-2025-10k-run',
		runnerId: 'r-johan',
		bib: '210',
		finishSeconds: 45 * 60 + 12,
		category: { gender: 'M', ageGroup: 'M50-59' },
		placementOverall: 17,
		placementCategory: 1,
		splits: splits10k(45 * 60 + 12),
		conditions: 'Sunny, 18°C',
		notes: 'Veteran category win.'
	},
	{
		id: 'res-8',
		raceId: 'rc-2026-10k-run',
		runnerId: 'r-johan',
		bib: '210',
		finishSeconds: 44 * 60 + 58,
		category: { gender: 'M', ageGroup: 'M50-59' },
		placementOverall: 14,
		placementCategory: 1,
		splits: splits10k(44 * 60 + 58),
		conditions: 'Overcast, 15°C',
		notes: ''
	},
	{
		id: 'res-9',
		raceId: 'rc-2025-kids',
		runnerId: 'r-mira',
		bib: 'K-44',
		finishSeconds: 6 * 60 + 12,
		category: { gender: 'F', ageGroup: 'U12' },
		placementOverall: 6,
		placementCategory: 3,
		conditions: 'Sunny, 18°C',
		notes: 'First medal!'
	},
	{
		id: 'res-10',
		raceId: 'rc-2026-kids',
		runnerId: 'r-mira',
		bib: 'K-44',
		finishSeconds: 5 * 60 + 48,
		category: { gender: 'F', ageGroup: 'U12' },
		placementOverall: 4,
		placementCategory: 2,
		conditions: 'Overcast, 15°C',
		notes: ''
	},
	{
		id: 'res-11',
		raceId: 'rc-2026-5k-run',
		runnerId: 'r-oskar',
		bib: '173',
		finishSeconds: 21 * 60 + 9,
		category: { gender: 'M', ageGroup: 'M20-29' },
		placementOverall: 8,
		placementCategory: 3,
		splits: splits5k(21 * 60 + 9),
		conditions: 'Overcast, 15°C',
		notes: 'Top 10 finish!'
	},
	{
		id: 'res-12',
		raceId: 'rc-2025-5k-run',
		runnerId: 'r-oskar',
		bib: '173',
		finishSeconds: 22 * 60 + 4,
		category: { gender: 'M', ageGroup: 'M20-29' },
		placementOverall: 12,
		placementCategory: 5,
		splits: splits5k(22 * 60 + 4),
		conditions: 'Sunny, 18°C',
		notes: ''
	}
];
