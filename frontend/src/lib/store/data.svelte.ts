import { browser } from '$app/environment';
import {
	events as seedEvents,
	races as seedRaces,
	registrations as seedRegistrations,
	runners as seedRunners
} from '$lib/mock/data';
import type { Race, RaceEvent, Registration, Runner } from '$lib/types';

const STORAGE_KEY = 'the-run.data';
const VERSION = 2;

type Snapshot = {
	v: number;
	runners: Runner[];
	events: RaceEvent[];
	races: Race[];
	registrations: Registration[];
};

function seed(): Snapshot {
	return {
		v: VERSION,
		runners: structuredClone(seedRunners),
		events: structuredClone(seedEvents),
		races: structuredClone(seedRaces),
		registrations: structuredClone(seedRegistrations)
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
			registrations: parsed.registrations ?? []
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

export type RegistrationUpdate = {
	id: string;
	status?: Registration['status'];
	finishSeconds?: number | null; // null clears
	bib?: string;
	category?: Registration['category'];
	splits?: Registration['splits'];
	conditions?: string;
	notes?: string;
};

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
	get registrations(): readonly Registration[] {
		return snap.registrations;
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
		snap.registrations = snap.registrations.filter((r) => r.runnerId !== id);
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
		snap.registrations = snap.registrations.filter((r) => !raceIdsToDrop.includes(r.raceId));
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
		snap.registrations = snap.registrations.filter((r) => r.raceId !== id);
		persist();
	},

	// Registrations
	upsertRegistration(input: Omit<Registration, 'id'> & { id?: string }): Registration {
		const id = input.id ?? nextId('reg-', snap.registrations);
		const idx = snap.registrations.findIndex((r) => r.id === id);
		const next: Registration = { ...input, id };
		if (idx >= 0) snap.registrations[idx] = next;
		else snap.registrations.push(next);
		persist();
		return next;
	},
	deleteRegistration(id: string): void {
		snap.registrations = snap.registrations.filter((r) => r.id !== id);
		persist();
	},
	bulkUpdateRegistrations(updates: RegistrationUpdate[]): Registration[] {
		const byId: Record<string, Registration> = {};
		for (const r of snap.registrations) byId[r.id] = r;
		const touched: Registration[] = [];
		for (const u of updates) {
			const existing = byId[u.id];
			if (!existing) continue;
			const next: Registration = { ...existing };
			if (u.status !== undefined) next.status = u.status;
			if (u.bib !== undefined) next.bib = u.bib || undefined;
			if (u.category !== undefined) next.category = u.category;
			if (u.splits !== undefined) next.splits = u.splits;
			if (u.conditions !== undefined) next.conditions = u.conditions || undefined;
			if (u.notes !== undefined) next.notes = u.notes || undefined;
			if (u.finishSeconds === null) next.finishSeconds = undefined;
			else if (u.finishSeconds !== undefined) next.finishSeconds = u.finishSeconds;
			byId[u.id] = next;
			touched.push(next);
		}
		snap.registrations = snap.registrations.map((r) => byId[r.id] ?? r);
		persist();
		return touched;
	},

	reset(): void {
		const s = seed();
		snap.runners = s.runners;
		snap.events = s.events;
		snap.races = s.races;
		snap.registrations = s.registrations;
		persist();
	}
};
