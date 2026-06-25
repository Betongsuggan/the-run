import { api } from '$lib/http';
import type { Race, RaceEvent, Registration, Runner } from '$lib/types';

type Snapshot = {
	runners: Runner[];
	events: RaceEvent[];
	races: Race[];
	registrations: Registration[];
};

const snap = $state<Snapshot>({
	runners: [],
	events: [],
	races: [],
	registrations: []
});

let hydratedAt: number | null = null;
let hydratePromise: Promise<void> | null = null;

export type RegistrationUpdate = {
	raceId: string;
	runnerId: string;
	status?: Registration['status'];
	finishSeconds?: number | null;
	bib?: string;
	category?: Registration['category'];
	splits?: Registration['splits'];
	conditions?: string;
	notes?: string;
};

async function fetchAll(): Promise<void> {
	const [runnersRes, eventsRes, racesRes, registrationsRes] = await Promise.all([
		api<{ runners: Runner[] }>('/runners'),
		api<{ events: RaceEvent[] }>('/events'),
		api<{ races: Race[] }>('/races'),
		api<{ registrations: Registration[] }>('/registrations')
	]);
	snap.runners = runnersRes.runners ?? [];
	snap.events = eventsRes.events ?? [];
	snap.races = racesRes.races ?? [];
	snap.registrations = registrationsRes.registrations ?? [];
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
	get registrations(): readonly Registration[] {
		return snap.registrations;
	},
	get isHydrated(): boolean {
		return hydratedAt !== null;
	},

	async hydrate(force = false): Promise<void> {
		if (hydratePromise && !force) return hydratePromise;
		if (hydratedAt !== null && !force) return;
		hydratePromise = (async () => {
			try {
				await fetchAll();
				hydratedAt = Date.now();
			} finally {
				hydratePromise = null;
			}
		})();
		return hydratePromise;
	},

	// ── Local cache mutation helpers — called by admin/api.ts after a
	//    successful backend write so the reactive UI stays in sync.

	upsertRunner(r: Runner): void {
		const i = snap.runners.findIndex((x) => x.id === r.id);
		if (i >= 0) snap.runners[i] = r;
		else snap.runners.push(r);
	},
	removeRunner(id: string): void {
		snap.runners = snap.runners.filter((r) => r.id !== id);
		snap.registrations = snap.registrations.filter((r) => r.runnerId !== id);
	},

	upsertEvent(e: RaceEvent): void {
		const i = snap.events.findIndex((x) => x.id === e.id);
		if (i >= 0) snap.events[i] = e;
		else snap.events.push(e);
	},
	removeEvent(id: string): void {
		const droppedRaces = snap.races.filter((r) => r.eventId === id).map((r) => r.id);
		snap.events = snap.events.filter((e) => e.id !== id);
		snap.races = snap.races.filter((r) => r.eventId !== id);
		snap.registrations = snap.registrations.filter((r) => !droppedRaces.includes(r.raceId));
	},

	upsertRace(race: Race): void {
		const i = snap.races.findIndex((x) => x.id === race.id);
		if (i >= 0) snap.races[i] = race;
		else snap.races.push(race);
	},
	removeRace(id: string): void {
		snap.races = snap.races.filter((r) => r.id !== id);
		snap.registrations = snap.registrations.filter((r) => r.raceId !== id);
	},

	upsertRegistration(reg: Registration): void {
		const i = snap.registrations.findIndex(
			(x) => x.raceId === reg.raceId && x.runnerId === reg.runnerId
		);
		if (i >= 0) snap.registrations[i] = reg;
		else snap.registrations.push(reg);
	},
	upsertRegistrations(regs: Registration[]): void {
		for (const r of regs) this.upsertRegistration(r);
	},
	removeRegistration(raceId: string, runnerId: string): void {
		snap.registrations = snap.registrations.filter(
			(r) => !(r.raceId === raceId && r.runnerId === runnerId)
		);
	}
};
