import { api } from '$lib/http';
import { dataStore, type RegistrationUpdate } from '$lib/store/data.svelte';
import type {
	AdminAccount,
	AdminInvitation,
	EmailTemplate,
	EmailTemplateRevision,
	Policy,
	PolicyRevision,
	Race,
	RaceEvent,
	Registration,
	Runner
} from '$lib/types';

// All admin endpoints are gated by the session cookie set by /auth/login +
// /auth/totp. The fetch helper sends `credentials: 'include'` so the cookie
// flows on cross-subdomain calls. See $lib/admin/auth.svelte.ts for the
// session state machine.

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

// ─── Privacy policies (admin CRUD) ──────────────────────────────────────

export type CreatePolicyInput = {
	kind: string;
	slug: string;
	effectiveFrom: string;
	bodySv: string;
	bodyEn: string;
};

export type EditPolicyInput = {
	bodySv: string;
	bodyEn: string;
	note?: string;
};

export async function adminListPolicies(): Promise<Policy[]> {
	const res = await api<{ policies: Policy[] }>('/admin/policies');
	return res.policies ?? [];
}

export async function adminCreatePolicy(input: CreatePolicyInput): Promise<Policy> {
	return api<Policy>('/admin/policies', { method: 'POST', body: input });
}

export async function adminEditPolicy(id: string, input: EditPolicyInput): Promise<Policy> {
	return api<Policy>(`/admin/policies/${encodeURIComponent(id)}`, {
		method: 'PUT',
		body: input
	});
}

export async function adminPublishPolicy(id: string): Promise<Policy> {
	return api<Policy>(`/admin/policies/${encodeURIComponent(id)}/publish`, { method: 'POST' });
}

export async function adminArchivePolicy(id: string): Promise<Policy> {
	return api<Policy>(`/admin/policies/${encodeURIComponent(id)}/archive`, { method: 'POST' });
}

export async function adminListPolicyRevisions(id: string): Promise<PolicyRevision[]> {
	const res = await api<{ revisions: PolicyRevision[] }>(
		`/admin/policies/${encodeURIComponent(id)}/revisions`
	);
	return res.revisions ?? [];
}

// ─── Email templates (admin CRUD) ───────────────────────────────────────

export type EditEmailTemplateInput = {
	subjectSv: string;
	bodySv: string;
	subjectEn: string;
	bodyEn: string;
	note?: string;
};

export async function adminListEmailTemplates(): Promise<EmailTemplate[]> {
	const res = await api<{ templates: EmailTemplate[] }>('/admin/email-templates');
	return res.templates ?? [];
}

export async function adminGetEmailTemplate(slug: string): Promise<EmailTemplate> {
	return api<EmailTemplate>(`/admin/email-templates/${encodeURIComponent(slug)}`);
}

export async function adminEditEmailTemplate(
	slug: string,
	input: EditEmailTemplateInput
): Promise<EmailTemplate> {
	return api<EmailTemplate>(`/admin/email-templates/${encodeURIComponent(slug)}`, {
		method: 'PUT',
		body: input
	});
}

export async function adminPublishEmailTemplate(slug: string): Promise<EmailTemplate> {
	return api<EmailTemplate>(`/admin/email-templates/${encodeURIComponent(slug)}/publish`, {
		method: 'POST'
	});
}

export async function adminArchiveEmailTemplate(slug: string): Promise<EmailTemplate> {
	return api<EmailTemplate>(`/admin/email-templates/${encodeURIComponent(slug)}/archive`, {
		method: 'POST'
	});
}

export async function adminListEmailTemplateRevisions(
	slug: string
): Promise<EmailTemplateRevision[]> {
	const res = await api<{ revisions: EmailTemplateRevision[] }>(
		`/admin/email-templates/${encodeURIComponent(slug)}/revisions`
	);
	return res.revisions ?? [];
}

// ─── Admin users + invitations ──────────────────────────────────────────

export async function adminListAdmins(): Promise<AdminAccount[]> {
	const res = await api<{ admins: AdminAccount[] }>('/admin/users');
	return res.admins ?? [];
}

export async function adminListInvitations(): Promise<AdminInvitation[]> {
	const res = await api<{ invitations: AdminInvitation[] }>('/admin/users/invitations');
	return res.invitations ?? [];
}

export type CreateInvitationInput = {
	email: string;
	locale?: 'sv' | 'en';
};

export async function adminCreateInvitation(
	input: CreateInvitationInput
): Promise<AdminInvitation> {
	return api<AdminInvitation>('/admin/users/invitations', {
		method: 'POST',
		body: input
	});
}

export async function adminRevokeInvitation(tokenId: string): Promise<void> {
	await api<void>(`/admin/users/invitations/${encodeURIComponent(tokenId)}`, {
		method: 'DELETE'
	});
}

// ─── Public admin accept endpoints ──────────────────────────────────────

export type AcceptPreview = {
	email: string;
	locale?: string;
	invitedByMail?: string;
};

export async function fetchAcceptPreview(token: string): Promise<AcceptPreview> {
	return api<AcceptPreview>(`/admin/accept/preview?token=${encodeURIComponent(token)}`);
}

export async function submitAcceptInvitation(
	token: string,
	password: string
): Promise<{ ok: boolean; email: string }> {
	return api<{ ok: boolean; email: string }>('/admin/accept', {
		method: 'POST',
		body: { token, password }
	});
}
