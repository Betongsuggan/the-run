import { PUBLIC_API_BASE_URL } from '$env/static/public';

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
