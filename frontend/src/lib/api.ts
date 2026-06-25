import { PUBLIC_API_BASE_URL } from '$env/static/public';
import { api } from '$lib/http';
import type { Race, RaceEvent, ResultExpanded, Runner } from '$lib/types';

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

export type RegisterForRaceInput = {
	name: string;
	email: string;
	dateOfBirth: string;
	gender: 'M' | 'F' | 'X';
	raceId: string;
	website?: string;
};

export type RegisterForRaceResult = {
	id: string;
	status: string;
};

export async function registerForRace(
	input: RegisterForRaceInput,
	fetchImpl: typeof fetch = fetch
): Promise<RegisterForRaceResult> {
	const res = await fetchImpl(`${PUBLIC_API_BASE_URL}/registrations`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ website: '', ...input })
	});
	if (!res.ok) {
		let detail = '';
		try {
			const body = await res.json();
			if (body && typeof body.detail === 'string') detail = `: ${body.detail}`;
		} catch {
			/* ignore */
		}
		throw new Error(`POST /registrations failed: HTTP ${res.status}${detail}`);
	}
	return (await res.json()) as RegisterForRaceResult;
}

export async function listRunners(): Promise<Runner[]> {
	const res = await api<{ runners: Runner[] }>('/runners');
	return res.runners ?? [];
}

export async function getRunner(id: string): Promise<Runner | undefined> {
	try {
		return await api<Runner>(`/runners/${encodeURIComponent(id)}`);
	} catch (err) {
		if (err instanceof Error && err.message.includes('HTTP 404')) return undefined;
		throw err;
	}
}

export async function listResultsForRunner(runnerId: string): Promise<ResultExpanded[]> {
	const res = await api<{ results: ResultExpanded[] }>(
		`/runners/${encodeURIComponent(runnerId)}/results`
	);
	return res.results ?? [];
}

export async function getResult(id: string): Promise<ResultExpanded | undefined> {
	try {
		return await api<ResultExpanded>(`/results/${encodeURIComponent(id)}`);
	} catch (err) {
		if (err instanceof Error && err.message.includes('HTTP 404')) return undefined;
		throw err;
	}
}

export async function listEvents(): Promise<RaceEvent[]> {
	const res = await api<{ events: RaceEvent[] }>('/events');
	return res.events ?? [];
}

export async function getEvent(id: string): Promise<RaceEvent | undefined> {
	try {
		return await api<RaceEvent>(`/events/${encodeURIComponent(id)}`);
	} catch (err) {
		if (err instanceof Error && err.message.includes('HTTP 404')) return undefined;
		throw err;
	}
}

export async function listRacesForEvent(eventId: string): Promise<Race[]> {
	const res = await api<{ races: Race[] }>(`/events/${encodeURIComponent(eventId)}/races`);
	return res.races ?? [];
}

export async function listResultsForRace(raceId: string): Promise<ResultExpanded[]> {
	const res = await api<{ results: ResultExpanded[] }>(
		`/races/${encodeURIComponent(raceId)}/results`
	);
	return res.results ?? [];
}
