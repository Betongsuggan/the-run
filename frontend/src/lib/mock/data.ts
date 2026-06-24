import type { Race, RaceEvent, Result, Runner } from '$lib/types';

export const runners: Runner[] = [
	{ id: 'r-birger', name: 'Birger Rydback', gender: 'M', birthYear: 1988 },
	{ id: 'r-luisa', name: 'Luisa Stock', gender: 'F', birthYear: 1991 },
	{ id: 'r-kalle', name: 'Kalle Torlén', gender: 'M', birthYear: 1975 },
	{ id: 'r-viktoria', name: 'Viktoria Torlén', gender: 'F', birthYear: 2014 }
];

export const events: RaceEvent[] = [
	{
		id: 'e-2025',
		name: 'Ingmarsöloppet 2025',
		year: 2025,
		date: '2025-06-14',
		location: 'Stockholm'
	},
	{
		id: 'e-2026',
		name: 'Ingmarsöloppet 2026',
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
		raceId: 'rc-2025-5k-walk',
		runnerId: 'r-birger',
		bib: '142',
		finishSeconds: 36 * 60 + 42,
		category: { gender: 'M', ageGroup: 'M30-39' },
		placementOverall: 7,
		placementCategory: 3,
		splits: splits5k(36 * 60 + 42),
		conditions: 'Sunny, 18°C, light wind',
		notes: 'Nice steady tempo all the way.'
	},
	{
		id: 'res-3',
		raceId: 'rc-2026-5k-walk',
		runnerId: 'r-birger',
		bib: '88',
		finishSeconds: 35 * 60 + 18,
		category: { gender: 'M', ageGroup: 'M30-39' },
		placementOverall: 5,
		placementCategory: 2,
		splits: splits5k(35 * 60 + 18),
		conditions: 'Overcast, 14°C',
		notes: 'New 5K walk PB.'
	},
	{
		id: 'res-6',
		raceId: 'rc-2026-10k-run',
		runnerId: 'r-luisa',
		bib: '301',
		finishSeconds: 53 * 60 + 24,
		category: { gender: 'F', ageGroup: 'F30-39' },
		placementOverall: 22,
		placementCategory: 5,
		splits: splits10k(53 * 60 + 24),
		conditions: 'Overcast, 15°C',
		notes: 'First 10K. Held back the first half and finished strong.'
	},
	{
		id: 'res-7',
		raceId: 'rc-2025-10k-run',
		runnerId: 'r-kalle',
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
		runnerId: 'r-kalle',
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
		runnerId: 'r-viktoria',
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
		runnerId: 'r-viktoria',
		bib: 'K-44',
		finishSeconds: 5 * 60 + 48,
		category: { gender: 'F', ageGroup: 'U12' },
		placementOverall: 4,
		placementCategory: 2,
		conditions: 'Overcast, 15°C',
		notes: ''
	}
];
