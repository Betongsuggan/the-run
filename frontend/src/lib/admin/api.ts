import { browser } from '$app/environment';
import { auth, type AdminUser } from '$lib/admin/auth.svelte';
import { dataStore, type RegistrationUpdate } from '$lib/store/data.svelte';
import type { Race, RaceEvent, Registration, Runner } from '$lib/types';

// ─── Mock backend store for users + invites ─────────────────────────────
// Persists alongside the data store. Real backend replaces this.

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

type Passwords = Record<string, string>; // email → password (mock only — backend hashes)

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

// ─── Auth ───────────────────────────────────────────────────────────────

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
	// Real backend would email a link here. Surface to the UI instead.
	console.info(
		`[mock] invite link for ${invite.email}: /admin/accept?token=${invite.token}`
	);
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

// ─── Runners CRUD ───────────────────────────────────────────────────────

export async function adminCreateRunner(input: Omit<Runner, 'id'>): Promise<Runner> {
	await delay();
	return dataStore.upsertRunner(input);
}

export async function adminUpdateRunner(runner: Runner): Promise<Runner> {
	await delay();
	return dataStore.upsertRunner(runner);
}

export async function adminDeleteRunner(id: string): Promise<void> {
	await delay();
	dataStore.deleteRunner(id);
}

// ─── Events CRUD ────────────────────────────────────────────────────────

export async function adminCreateEvent(input: Omit<RaceEvent, 'id'>): Promise<RaceEvent> {
	await delay();
	return dataStore.upsertEvent(input);
}

export async function adminUpdateEvent(event: RaceEvent): Promise<RaceEvent> {
	await delay();
	return dataStore.upsertEvent(event);
}

export async function adminDeleteEvent(id: string): Promise<void> {
	await delay();
	dataStore.deleteEvent(id);
}

// ─── Races CRUD ─────────────────────────────────────────────────────────

export async function adminCreateRace(input: Omit<Race, 'id'>): Promise<Race> {
	await delay();
	return dataStore.upsertRace(input);
}

export async function adminUpdateRace(race: Race): Promise<Race> {
	await delay();
	return dataStore.upsertRace(race);
}

export async function adminDeleteRace(id: string): Promise<void> {
	await delay();
	dataStore.deleteRace(id);
}

// ─── Registrations + bulk results ───────────────────────────────────────

export async function adminListRegistrationsForRace(raceId: string): Promise<Registration[]> {
	await delay(20);
	return dataStore.registrations.filter((r) => r.raceId === raceId).map((r) => ({ ...r }));
}

export type CreateRegistrationInput = {
	raceId: string;
	runnerId: string;
	bib?: string;
	category?: Registration['category'];
};

export async function adminCreateRegistration(input: CreateRegistrationInput): Promise<Registration> {
	await delay();
	const existing = dataStore.registrations.find(
		(r) => r.raceId === input.raceId && r.runnerId === input.runnerId
	);
	if (existing) throw new Error('This runner is already registered for this race.');
	const runner = dataStore.runners.find((r) => r.id === input.runnerId);
	if (!runner) throw new Error('Runner not found.');
	const category = input.category ?? { gender: runner.gender };
	return dataStore.upsertRegistration({
		raceId: input.raceId,
		runnerId: input.runnerId,
		bib: input.bib?.trim() || undefined,
		category,
		status: 'pending'
	});
}

export async function adminDeleteRegistration(id: string): Promise<void> {
	await delay();
	dataStore.deleteRegistration(id);
}

export async function adminBulkUpdateRegistrations(
	updates: RegistrationUpdate[]
): Promise<Registration[]> {
	await delay();
	const normalized: RegistrationUpdate[] = updates.map((u) => {
		const status = u.status;
		if (status === 'finished') {
			if (u.finishSeconds === null || u.finishSeconds === undefined) {
				throw new Error('A finished registration requires a finish time.');
			}
			return u;
		}
		if (status === 'dnf' || status === 'dns' || status === 'pending') {
			return { ...u, finishSeconds: null };
		}
		return u;
	});
	return dataStore.bulkUpdateRegistrations(normalized);
}

// Exposed so the UI can hint at the seed account in mock mode.
export const MOCK_SEED_ADMIN = {
	email: SEED_ADMIN_EMAIL,
	password: SEED_ADMIN_PASSWORD
};
