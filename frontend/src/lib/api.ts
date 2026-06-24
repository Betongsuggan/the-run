import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { dataStore } from '$lib/store/data.svelte';
import type { Race, RaceEvent, Result, ResultExpanded, Runner } from '$lib/types';

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

const expand = (result: Result): ResultExpanded | null => {
	const race = dataStore.races.find((r) => r.id === result.raceId);
	if (!race) return null;
	const event = dataStore.events.find((e) => e.id === race.eventId);
	if (!event) return null;
	const runner = dataStore.runners.find((rn) => rn.id === result.runnerId);
	if (!runner) return null;
	return { ...result, race, event, runner };
};

export async function listRunners(): Promise<Runner[]> {
	return Promise.resolve([...dataStore.runners]);
}

export async function getRunner(id: string): Promise<Runner | undefined> {
	return Promise.resolve(dataStore.runners.find((r) => r.id === id));
}

export async function listResultsForRunner(runnerId: string): Promise<ResultExpanded[]> {
	const expanded = dataStore.results
		.filter((r) => r.runnerId === runnerId)
		.map(expand)
		.filter((r): r is ResultExpanded => r !== null)
		.sort((a, b) => (a.event.date < b.event.date ? 1 : -1));
	return Promise.resolve(expanded);
}

export async function getResult(id: string): Promise<ResultExpanded | undefined> {
	const result = dataStore.results.find((r) => r.id === id);
	if (!result) return undefined;
	const expanded = expand(result);
	return Promise.resolve(expanded ?? undefined);
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
	const expanded = dataStore.results
		.filter((r) => r.raceId === raceId)
		.map(expand)
		.filter((r): r is ResultExpanded => r !== null)
		.sort((a, b) => a.finishSeconds - b.finishSeconds);
	return Promise.resolve(expanded);
}
