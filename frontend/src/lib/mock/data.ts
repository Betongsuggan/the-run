import type { Race, RaceEvent, Registration, Runner } from '$lib/types';

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

export const registrations: Registration[] = [
	{
		id: 'reg-1',
		raceId: 'rc-2025-5k-walk',
		runnerId: 'r-birger',
		bib: '142',
		category: { gender: 'M', ageGroup: 'M30-39' },
		status: 'finished',
		finishSeconds: 36 * 60 + 42,
		splits: splits5k(36 * 60 + 42),
		conditions: 'Sunny, 18°C, light wind',
		notes: 'Nice steady tempo all the way.'
	},
	{
		id: 'reg-3',
		raceId: 'rc-2026-5k-walk',
		runnerId: 'r-birger',
		bib: '88',
		category: { gender: 'M', ageGroup: 'M30-39' },
		status: 'finished',
		finishSeconds: 35 * 60 + 18,
		splits: splits5k(35 * 60 + 18),
		conditions: 'Overcast, 14°C',
		notes: 'New 5K walk PB.'
	},
	{
		id: 'reg-6',
		raceId: 'rc-2026-10k-run',
		runnerId: 'r-luisa',
		bib: '301',
		category: { gender: 'F', ageGroup: 'F30-39' },
		status: 'finished',
		finishSeconds: 53 * 60 + 24,
		splits: splits10k(53 * 60 + 24),
		conditions: 'Overcast, 15°C',
		notes: 'First 10K. Held back the first half and finished strong.'
	},
	{
		id: 'reg-7',
		raceId: 'rc-2025-10k-run',
		runnerId: 'r-kalle',
		bib: '210',
		category: { gender: 'M', ageGroup: 'M50-59' },
		status: 'finished',
		finishSeconds: 45 * 60 + 12,
		splits: splits10k(45 * 60 + 12),
		conditions: 'Sunny, 18°C',
		notes: 'Veteran category win.'
	},
	{
		id: 'reg-8',
		raceId: 'rc-2026-10k-run',
		runnerId: 'r-kalle',
		bib: '210',
		category: { gender: 'M', ageGroup: 'M50-59' },
		status: 'finished',
		finishSeconds: 44 * 60 + 58,
		conditions: 'Overcast, 15°C'
	},
	{
		id: 'reg-9',
		raceId: 'rc-2025-kids',
		runnerId: 'r-viktoria',
		bib: 'K-44',
		category: { gender: 'F', ageGroup: 'U12' },
		status: 'finished',
		finishSeconds: 6 * 60 + 12,
		conditions: 'Sunny, 18°C',
		notes: 'First medal!'
	},
	{
		id: 'reg-10',
		raceId: 'rc-2026-kids',
		runnerId: 'r-viktoria',
		bib: 'K-44',
		category: { gender: 'F', ageGroup: 'U12' },
		status: 'finished',
		finishSeconds: 5 * 60 + 48,
		conditions: 'Overcast, 15°C'
	}
];
