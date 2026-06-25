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
};

export type DSRRunner = {
	id: string;
	name: string;
	gender: string;
	birthDate: string;
	createdAt: string;
	publicResultsConsent?: DSRConsent;
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

export type DSRMe = {
	account: DSRAccount;
	runners: DSRRunner[];
	registrations: DSRRegistration[];
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
