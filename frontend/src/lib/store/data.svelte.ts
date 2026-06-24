import { browser } from '$app/environment';
import {
	events as seedEvents,
	races as seedRaces,
	results as seedResults,
	runners as seedRunners
} from '$lib/mock/data';
import type { Race, RaceEvent, Result, Runner } from '$lib/types';

const STORAGE_KEY = 'the-run.data';
const VERSION = 1;

type Snapshot = {
	v: number;
	runners: Runner[];
	events: RaceEvent[];
	races: Race[];
	results: Result[];
};

function seed(): Snapshot {
	return {
		v: VERSION,
		runners: structuredClone(seedRunners),
		events: structuredClone(seedEvents),
		races: structuredClone(seedRaces),
		results: structuredClone(seedResults)
	};
}

function load(): Snapshot {
	if (!browser) return seed();
	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) return seed();
		const parsed = JSON.parse(raw) as Partial<Snapshot>;
		if (parsed.v !== VERSION) return seed();
		return {
			v: VERSION,
			runners: parsed.runners ?? [],
			events: parsed.events ?? [],
			races: parsed.races ?? [],
			results: parsed.results ?? []
		};
	} catch {
		return seed();
	}
}

const snap = $state<Snapshot>(load());

function persist(): void {
	if (!browser) return;
	try {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(snap));
	} catch {
		/* quota / unavailable — silently ignore */
	}
}

function nextId(prefix: string, existing: ReadonlyArray<{ id: string }>): string {
	const used = new Set(existing.map((x) => x.id));
	let n = existing.length + 1;
	let id = `${prefix}${n}`;
	while (used.has(id)) {
		n += 1;
		id = `${prefix}${n}`;
	}
	return id;
}

export const dataStore = {
	get runners(): readonly Runner[] {
		return snap.runners;
	},
	get events(): readonly RaceEvent[] {
		return snap.events;
	},
	get races(): readonly Race[] {
		return snap.races;
	},
	get results(): readonly Result[] {
		return snap.results;
	},

	// Runners
	upsertRunner(input: Omit<Runner, 'id'> & { id?: string }): Runner {
		const id = input.id ?? nextId('r-', snap.runners);
		const idx = snap.runners.findIndex((r) => r.id === id);
		const next: Runner = { ...input, id };
		if (idx >= 0) snap.runners[idx] = next;
		else snap.runners.push(next);
		persist();
		return next;
	},
	deleteRunner(id: string): void {
		snap.runners = snap.runners.filter((r) => r.id !== id);
		snap.results = snap.results.filter((r) => r.runnerId !== id);
		persist();
	},

	// Events
	upsertEvent(input: Omit<RaceEvent, 'id'> & { id?: string }): RaceEvent {
		const id = input.id ?? nextId('e-', snap.events);
		const idx = snap.events.findIndex((e) => e.id === id);
		const next: RaceEvent = { ...input, id };
		if (idx >= 0) snap.events[idx] = next;
		else snap.events.push(next);
		persist();
		return next;
	},
	deleteEvent(id: string): void {
		const raceIdsToDrop = snap.races.filter((r) => r.eventId === id).map((r) => r.id);
		snap.events = snap.events.filter((e) => e.id !== id);
		snap.races = snap.races.filter((r) => r.eventId !== id);
		snap.results = snap.results.filter((r) => !raceIdsToDrop.includes(r.raceId));
		persist();
	},

	// Races
	upsertRace(input: Omit<Race, 'id'> & { id?: string }): Race {
		const id = input.id ?? nextId('rc-', snap.races);
		const idx = snap.races.findIndex((r) => r.id === id);
		const next: Race = { ...input, id };
		if (idx >= 0) snap.races[idx] = next;
		else snap.races.push(next);
		persist();
		return next;
	},
	deleteRace(id: string): void {
		snap.races = snap.races.filter((r) => r.id !== id);
		snap.results = snap.results.filter((r) => r.raceId !== id);
		persist();
	},

	// Results
	upsertResult(input: Omit<Result, 'id'> & { id?: string }): Result {
		const id = input.id ?? nextId('res-', snap.results);
		const idx = snap.results.findIndex((r) => r.id === id);
		const next: Result = { ...input, id };
		if (idx >= 0) snap.results[idx] = next;
		else snap.results.push(next);
		persist();
		return next;
	},
	deleteResult(id: string): void {
		snap.results = snap.results.filter((r) => r.id !== id);
		persist();
	},

	reset(): void {
		const s = seed();
		snap.runners = s.runners;
		snap.events = s.events;
		snap.races = s.races;
		snap.results = s.results;
		persist();
	}
};
