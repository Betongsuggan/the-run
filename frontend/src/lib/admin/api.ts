import { browser } from '$app/environment';
import { api } from '$lib/http';
import { auth, type AdminUser } from '$lib/admin/auth.svelte';
import { dataStore, type RegistrationUpdate } from '$lib/store/data.svelte';
import type { Race, RaceEvent, Registration, Runner } from '$lib/types';

// ─── Auth — STILL MOCKED ────────────────────────────────────────────────
// TODO: backend auth. The login/invite/users surface persists in localStorage
// until we wire Cognito (or custom email+password) + JWT verification on the
// API. See plan: "Open / deferred — Auth" in the i-would-like-to-fluttering
// plan file. Admin write endpoints below ARE the real backend, but they are
// currently unauthenticated — do not expose this to the public internet.

const USERS_KEY = 'the-run.admin-users';
const INVITES_KEY = 'the-run.admin-invites';
const PASSWORDS_KEY = 'the-run.admin-passwords';

export type AdminInvite = {
	id: string;
	email: string;
	token: string;
	createdAt: string;
	createdByEmail: string;
	acceptedAt?: string;
};

type Passwords = Record<string, string>;

const SEED_ADMIN_EMAIL = 'admin@ingmarsoloppet.se';
const SEED_ADMIN_PASSWORD = 'changeme';

function readUsers(): AdminUser[] {
	if (!browser) return [];
	try {
		const raw = window.localStorage.getItem(USERS_KEY);
		if (raw) return JSON.parse(raw) as AdminUser[];
	} catch {
		/* fall through */
	}
	const seed: AdminUser = {
		id: 'u-admin',
		email: SEED_ADMIN_EMAIL,
		createdAt: new Date().toISOString()
	};
	writeUsers([seed]);
	return [seed];
}

function writeUsers(users: AdminUser[]): void {
	if (!browser) return;
	window.localStorage.setItem(USERS_KEY, JSON.stringify(users));
}

function readPasswords(): Passwords {
	if (!browser) return {};
	try {
		const raw = window.localStorage.getItem(PASSWORDS_KEY);
		if (raw) return JSON.parse(raw) as Passwords;
	} catch {
		/* fall through */
	}
	const seed: Passwords = { [SEED_ADMIN_EMAIL]: SEED_ADMIN_PASSWORD };
	writePasswords(seed);
	return seed;
}

function writePasswords(p: Passwords): void {
	if (!browser) return;
	window.localStorage.setItem(PASSWORDS_KEY, JSON.stringify(p));
}

function readInvites(): AdminInvite[] {
	if (!browser) return [];
	try {
		const raw = window.localStorage.getItem(INVITES_KEY);
		if (raw) return JSON.parse(raw) as AdminInvite[];
	} catch {
		/* fall through */
	}
	return [];
}

function writeInvites(invites: AdminInvite[]): void {
	if (!browser) return;
	window.localStorage.setItem(INVITES_KEY, JSON.stringify(invites));
}

function makeToken(): string {
	if (browser && typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
		return crypto.randomUUID();
	}
	return `tok-${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
}

function delay(ms = 120): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function login(email: string, password: string): Promise<AdminUser> {
	await delay();
	const users = readUsers();
	const passwords = readPasswords();
	const user = users.find((u) => u.email.toLowerCase() === email.toLowerCase());
	if (!user) throw new Error('No user with that email.');
	if (passwords[user.email] !== password) throw new Error('Incorrect password.');
	auth.setSession(makeToken(), user);
	return user;
}

export async function logout(): Promise<void> {
	await delay();
	auth.clear();
}

export async function inviteUser(email: string): Promise<AdminInvite> {
	await delay();
	if (!auth.user) throw new Error('Not authenticated.');
	const trimmed = email.trim().toLowerCase();
	if (!trimmed) throw new Error('Email is required.');
	const users = readUsers();
	if (users.find((u) => u.email.toLowerCase() === trimmed)) {
		throw new Error('A user with that email already exists.');
	}
	const invites = readInvites();
	if (invites.find((i) => !i.acceptedAt && i.email.toLowerCase() === trimmed)) {
		throw new Error('An invite is already pending for that email.');
	}
	const invite: AdminInvite = {
		id: `inv-${Date.now()}`,
		email: trimmed,
		token: makeToken(),
		createdAt: new Date().toISOString(),
		createdByEmail: auth.user.email
	};
	writeInvites([...invites, invite]);
	console.info(`[mock] invite link for ${invite.email}: /admin/accept?token=${invite.token}`);
	return invite;
}

export async function listInvites(): Promise<AdminInvite[]> {
	await delay(20);
	return [...readInvites()];
}

export async function revokeInvite(id: string): Promise<void> {
	await delay();
	writeInvites(readInvites().filter((i) => i.id !== id));
}

export async function listUsers(): Promise<AdminUser[]> {
	await delay(20);
	return [...readUsers()];
}

export async function deleteUser(id: string): Promise<void> {
	await delay();
	if (!auth.user) throw new Error('Not authenticated.');
	if (auth.user.id === id) throw new Error("You can't delete your own account.");
	const users = readUsers();
	const target = users.find((u) => u.id === id);
	if (!target) return;
	writeUsers(users.filter((u) => u.id !== id));
	const passwords = readPasswords();
	delete passwords[target.email];
	writePasswords(passwords);
}

export type AcceptInviteInput = { token: string; password: string };

export async function acceptInvite({ token, password }: AcceptInviteInput): Promise<AdminUser> {
	await delay();
	if (!password || password.length < 6) {
		throw new Error('Password must be at least 6 characters.');
	}
	const invites = readInvites();
	const invite = invites.find((i) => i.token === token);
	if (!invite) throw new Error('Invalid or expired invite.');
	if (invite.acceptedAt) throw new Error('This invite has already been used.');

	const user: AdminUser = {
		id: `u-${Date.now()}`,
		email: invite.email,
		createdAt: new Date().toISOString()
	};
	writeUsers([...readUsers(), user]);

	const passwords = readPasswords();
	passwords[user.email] = password;
	writePasswords(passwords);

	writeInvites(
		invites.map((i) => (i.id === invite.id ? { ...i, acceptedAt: new Date().toISOString() } : i))
	);

	auth.setSession(makeToken(), user);
	return user;
}

export const MOCK_SEED_ADMIN = {
	email: SEED_ADMIN_EMAIL,
	password: SEED_ADMIN_PASSWORD
};

// ─── Runners CRUD ───────────────────────────────────────────────────────

export async function adminCreateRunner(input: Omit<Runner, 'id'>): Promise<Runner> {
	const res = await api<Runner>('/runners', { method: 'POST', body: input });
	dataStore.upsertRunner(res);
	return res;
}

export async function adminUpdateRunner(runner: Runner): Promise<Runner> {
	const { id, ...body } = runner;
	const res = await api<Runner>(`/runners/${encodeURIComponent(id)}`, { method: 'PUT', body });
	dataStore.upsertRunner(res);
	return res;
}

export async function adminDeleteRunner(id: string): Promise<void> {
	await api<void>(`/runners/${encodeURIComponent(id)}`, { method: 'DELETE' });
	dataStore.removeRunner(id);
}

// ─── Events CRUD ────────────────────────────────────────────────────────

export async function adminCreateEvent(input: Omit<RaceEvent, 'id'>): Promise<RaceEvent> {
	const res = await api<RaceEvent>('/events', { method: 'POST', body: input });
	dataStore.upsertEvent(res);
	return res;
}

export async function adminUpdateEvent(event: RaceEvent): Promise<RaceEvent> {
	const { id, ...body } = event;
	const res = await api<RaceEvent>(`/events/${encodeURIComponent(id)}`, { method: 'PUT', body });
	dataStore.upsertEvent(res);
	return res;
}

export async function adminDeleteEvent(id: string): Promise<void> {
	await api<void>(`/events/${encodeURIComponent(id)}`, { method: 'DELETE' });
	dataStore.removeEvent(id);
}

// ─── Races CRUD ─────────────────────────────────────────────────────────

export async function adminCreateRace(input: Omit<Race, 'id'>): Promise<Race> {
	const res = await api<Race>('/races', { method: 'POST', body: input });
	dataStore.upsertRace(res);
	return res;
}

export async function adminUpdateRace(race: Race): Promise<Race> {
	const { id, ...body } = race;
	const res = await api<Race>(`/races/${encodeURIComponent(id)}`, { method: 'PUT', body });
	dataStore.upsertRace(res);
	return res;
}

export async function adminDeleteRace(id: string): Promise<void> {
	await api<void>(`/races/${encodeURIComponent(id)}`, { method: 'DELETE' });
	dataStore.removeRace(id);
}

// ─── Registrations + bulk results ───────────────────────────────────────

export async function adminListRegistrationsForRace(raceId: string): Promise<Registration[]> {
	const res = await api<{ registrations: Registration[] }>(
		`/races/${encodeURIComponent(raceId)}/registrations`
	);
	const list = res.registrations ?? [];
	dataStore.upsertRegistrations(list);
	return list;
}

export type CreateRegistrationInput = {
	raceId: string;
	runnerId: string;
	bib?: string;
	category?: Registration['category'];
};

export async function adminCreateRegistration(
	input: CreateRegistrationInput
): Promise<Registration> {
	const res = await api<Registration>('/registrations/admin', { method: 'POST', body: input });
	dataStore.upsertRegistration(res);
	return res;
}

export async function adminDeleteRegistration(raceId: string, runnerId: string): Promise<void> {
	await api<void>(
		`/races/${encodeURIComponent(raceId)}/registrations/${encodeURIComponent(runnerId)}`,
		{ method: 'DELETE' }
	);
	dataStore.removeRegistration(raceId, runnerId);
}

// Maps the frontend's "finishSeconds: null clears it" convention onto the
// backend's explicit `clearFinish` flag.
function toUpdatePayload(u: RegistrationUpdate) {
	const body: Record<string, unknown> = {
		raceId: u.raceId,
		runnerId: u.runnerId
	};
	if (u.status !== undefined) body.status = u.status;
	if (u.bib !== undefined) body.bib = u.bib;
	if (u.category !== undefined) body.category = u.category;
	if (u.conditions !== undefined) body.conditions = u.conditions;
	if (u.notes !== undefined) body.notes = u.notes;
	if (u.splits !== undefined) body.splits = u.splits;
	if (u.finishSeconds === null) body.clearFinish = true;
	else if (u.finishSeconds !== undefined) body.finishSeconds = u.finishSeconds;
	return body;
}

export async function adminBulkUpdateRegistrations(
	updates: RegistrationUpdate[]
): Promise<Registration[]> {
	if (updates.length === 0) return [];
	const res = await api<{ registrations: Registration[] }>('/registrations/bulk', {
		method: 'PATCH',
		body: { updates: updates.map(toUpdatePayload) }
	});
	const list = res.registrations ?? [];
	dataStore.upsertRegistrations(list);
	return list;
}
