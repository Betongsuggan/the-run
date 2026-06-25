import { PUBLIC_API_BASE_URL } from '$env/static/public';

export type RequestOptions = {
	method?: string;
	body?: unknown;
	fetch?: typeof fetch;
};

/**
 * Thin fetch wrapper for the Huma-backed API. Prepends the base URL, serializes
 * JSON bodies, and unwraps Huma's `{detail, errors[]}` error shape so callers
 * see a meaningful Error message.
 */
export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
	const fetchImpl = opts.fetch ?? fetch;
	const init: RequestInit = {
		method: opts.method ?? 'GET',
		headers: opts.body !== undefined ? { 'Content-Type': 'application/json' } : undefined
	};
	if (opts.body !== undefined) init.body = JSON.stringify(opts.body);

	const res = await fetchImpl(`${PUBLIC_API_BASE_URL}${path}`, init);
	if (!res.ok) {
		let detail = '';
		try {
			const body = (await res.json()) as { detail?: string };
			if (body && typeof body.detail === 'string') detail = `: ${body.detail}`;
		} catch {
			/* no JSON body */
		}
		throw new Error(`${init.method} ${path} failed: HTTP ${res.status}${detail}`);
	}
	if (res.status === 204) return undefined as T;
	return (await res.json()) as T;
}
