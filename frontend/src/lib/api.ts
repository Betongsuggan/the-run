import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { dataStore } from '$lib/store/data.svelte';
import type {
	Race,
	RaceEvent,
	Registration,
	Result,
	ResultExpanded,
	Runner
} from '$lib/types';

export type Hello = {
	message: string;
	timestamp: string;
};

export async function getHello(fetchImpl: typeof fetch = fetch): Promise<Hello> {
	const res = await fetchImpl(`${PUBLIC_API_BASE_URL}/hello`);
	if (!res.ok) {
		throw new Error(`GET /hello failed: HTTP ${res.status}`);
	}
	return (await res.json()) as Hello;
}

type FinishedRegistration = Registration & { finishSeconds: number };

function isFinished(reg: Registration): reg is FinishedRegistration {
	return reg.status === 'finished' && typeof reg.finishSeconds === 'number';
}

function categoryKey(cat: Registration['category']): string {
	return `${cat?.gender ?? ''}|${cat?.ageGroup ?? ''}`;
}

function toResult(reg: FinishedRegistration, placementOverall: number, placementCategory: number): Result {
	return {
		id: reg.id,
		raceId: reg.raceId,
		runnerId: reg.runnerId,
		bib: reg.bib ?? '',
		finishSeconds: reg.finishSeconds,
		category: reg.category ?? {},
		placementOverall,
		placementCategory,
		splits: reg.splits,
		conditions: reg.conditions,
		notes: reg.notes
	};
}

// Compute placement-enriched Results for every finished registration in a race.
function computeResultsForRace(raceId: string): Result[] {
	const finished = dataStore.registrations.filter(
		(r): r is FinishedRegistration => r.raceId === raceId && isFinished(r)
	);
	const sorted = [...finished].sort((a, b) => a.finishSeconds - b.finishSeconds);

	const catCounter: Record<string, number> = {};
	return sorted.map((reg, idx) => {
		const key = categoryKey(reg.category);
		catCounter[key] = (catCounter[key] ?? 0) + 1;
		return toResult(reg, idx + 1, catCounter[key]);
	});
}

function expand(result: Result): ResultExpanded | null {
	const race = dataStore.races.find((r) => r.id === result.raceId);
	if (!race) return null;
	const event = dataStore.events.find((e) => e.id === race.eventId);
	if (!event) return null;
	const runner = dataStore.runners.find((rn) => rn.id === result.runnerId);
	if (!runner) return null;
	return { ...result, race, event, runner };
}

export async function listRunners(): Promise<Runner[]> {
	return Promise.resolve([...dataStore.runners]);
}

export async function getRunner(id: string): Promise<Runner | undefined> {
	return Promise.resolve(dataStore.runners.find((r) => r.id === id));
}

export async function listResultsForRunner(runnerId: string): Promise<ResultExpanded[]> {
	const raceIds: string[] = [];
	for (const r of dataStore.registrations) {
		if (r.runnerId === runnerId && isFinished(r) && !raceIds.includes(r.raceId)) {
			raceIds.push(r.raceId);
		}
	}
	const expanded: ResultExpanded[] = [];
	for (const raceId of raceIds) {
		const results = computeResultsForRace(raceId).filter((r) => r.runnerId === runnerId);
		for (const r of results) {
			const e = expand(r);
			if (e) expanded.push(e);
		}
	}
	expanded.sort((a, b) => (a.event.date < b.event.date ? 1 : -1));
	return Promise.resolve(expanded);
}

export async function getResult(id: string): Promise<ResultExpanded | undefined> {
	const reg = dataStore.registrations.find((r) => r.id === id);
	if (!reg || !isFinished(reg)) return undefined;
	const results = computeResultsForRace(reg.raceId);
	const result = results.find((r) => r.id === id);
	if (!result) return undefined;
	return Promise.resolve(expand(result) ?? undefined);
}

export async function listEvents(): Promise<RaceEvent[]> {
	return Promise.resolve([...dataStore.events]);
}

export async function getEvent(id: string): Promise<RaceEvent | undefined> {
	return Promise.resolve(dataStore.events.find((e) => e.id === id));
}

export async function listRacesForEvent(eventId: string): Promise<Race[]> {
	return Promise.resolve(dataStore.races.filter((r) => r.eventId === eventId));
}

export async function listResultsForRace(raceId: string): Promise<ResultExpanded[]> {
	const expanded = computeResultsForRace(raceId)
		.map(expand)
		.filter((r): r is ResultExpanded => r !== null);
	return Promise.resolve(expanded);
}
