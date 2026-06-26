// DSR (Data Subject Rights) client for /my-data. Mirrors the Go-side
// /dsr/* endpoints in backend/internal/api/dsr.go. Slice A1.1.1: read +
// export only; edit/erase/restore follow in later slices.

import { api } from '$lib/http';
import { PUBLIC_API_BASE_URL } from '$env/static/public';

export type DSRConsent = {
	granted: boolean;
	at: string;
	policyVersion: string;
};

export type DSRAccount = {
	id: string;
	email: string;
	locale?: string;
	createdAt: string;
	marketingConsent?: DSRConsent;
	// ISO timestamp when the account will be permanently purged. Set
	// only when the account is in the 30-day soft-delete grace window.
	deletionPendingUntil?: string;
	// ISO timestamp when this account would be flagged for inactivity-
	// based deletion if no further activity occurs. Always set on active
	// accounts (resets on every login / new registration / finished race).
	inactivityDeletionAt?: string;
};

export type DSRRunnerRetention = {
	// "kept-indefinitely" | "anonymizes-at" | "no-timer"
	policy: 'kept-indefinitely' | 'anonymizes-at' | 'no-timer';
	anonymizesAt?: string; // YYYY-MM-DD
	lastFinishedRace?: string; // YYYY-MM-DD
};

export type DSRRunner = {
	id: string;
	name: string;
	gender: string;
	birthDate: string;
	createdAt: string;
	publicResultsConsent?: DSRConsent;
	retention: DSRRunnerRetention;
};

export type DSRRegistration = {
	id: string;
	raceId: string;
	runnerId: string;
	status: string;
	bib?: string;
	finishSeconds?: number;
	createdAt: string;
};

export type DSRAuditRow = {
	at: string;
	action: string;
	actor: string;
	targetType?: string;
	targetId?: string;
	summary?: string;
};

export type DSRMe = {
	account: DSRAccount;
	runners: DSRRunner[];
	registrations: DSRRegistration[];
	recentAudit?: DSRAuditRow[];
};

export type DSRSession = {
	id: string;
	email: string;
};

export async function dsrRequestLink(email: string): Promise<void> {
	await api<{ ok: boolean }>('/dsr/request-link', { method: 'POST', body: { email } });
}

export async function dsrVerify(token: string): Promise<DSRSession> {
	return api<DSRSession>('/dsr/verify', { method: 'POST', body: { token } });
}

export async function dsrLogout(): Promise<void> {
	await api<{ ok: boolean }>('/dsr/logout', { method: 'POST' });
}

export async function dsrMe(): Promise<DSRMe> {
	return api<DSRMe>('/dsr/me');
}

export type DSRConsentUpdate = { publicResults?: boolean };

export type DSRPatchConsentsInput = {
	marketing?: boolean;
	perRunner?: Record<string, DSRConsentUpdate>;
};

export async function dsrPatchConsents(body: DSRPatchConsentsInput): Promise<DSRMe> {
	return api<DSRMe>('/dsr/me/consents', { method: 'PATCH', body });
}

export type DSRPatchRunnerInput = {
	name?: string;
	gender?: 'M' | 'F' | 'X';
	dateOfBirth?: string;
};

export async function dsrPatchRunner(id: string, body: DSRPatchRunnerInput): Promise<DSRMe> {
	return api<DSRMe>(`/dsr/me/runners/${encodeURIComponent(id)}`, { method: 'PATCH', body });
}

export async function dsrPatchLocale(locale: 'sv' | 'en'): Promise<DSRMe> {
	return api<DSRMe>('/dsr/me/locale', { method: 'PATCH', body: { locale } });
}

export async function dsrRequestEmailChange(newEmail: string): Promise<void> {
	await api<{ ok: boolean }>('/dsr/me/email/request-change', {
		method: 'POST',
		body: { newEmail }
	});
}

export async function dsrConfirmEmailChange(token: string): Promise<{ email: string }> {
	return api<{ email: string }>('/dsr/me/email/confirm', { method: 'POST', body: { token } });
}

export async function dsrEraseRunner(id: string): Promise<DSRMe> {
	return api<DSRMe>(`/dsr/me/runners/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function dsrEraseAccount(): Promise<{ ok: boolean }> {
	return api<{ ok: boolean }>('/dsr/me', { method: 'DELETE' });
}

export async function dsrRestore(token: string): Promise<DSRSession> {
	return api<DSRSession>('/dsr/me/restore', { method: 'POST', body: { token } });
}

// Session-authenticated counterpart to dsrRestore — used by the dashboard
// "cancel deletion" banner when the user is already logged in.
export async function dsrCancelDeletion(): Promise<DSRMe> {
	return api<DSRMe>('/dsr/me/cancel-deletion', { method: 'POST' });
}

// dsrExportDownload bypasses the standard api() wrapper because it doesn't
// return JSON to the caller — it triggers a browser download. We do the
// fetch directly so we can read the response body as a blob and use a
// hidden anchor + URL.createObjectURL to save it.
export async function dsrExportDownload(): Promise<void> {
	const res = await fetch(`${PUBLIC_API_BASE_URL}/dsr/me/export`, {
		credentials: 'include'
	});
	if (!res.ok) {
		throw new Error(`GET /dsr/me/export failed: HTTP ${res.status}`);
	}
	const blob = await res.blob();
	const filename =
		parseFilenameFromDisposition(res.headers.get('Content-Disposition')) ?? 'my-data.json';
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}

function parseFilenameFromDisposition(header: string | null): string | null {
	if (!header) return null;
	const m = /filename="([^"]+)"/.exec(header);
	return m?.[1] ?? null;
}
