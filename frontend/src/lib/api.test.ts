import { describe, it, expect, vi } from 'vitest';
import { getHello } from './api';

describe('getHello', () => {
	it('parses a successful response', async () => {
		const fetchImpl = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({ message: 'hi', timestamp: '2026-01-01T00:00:00Z' })
		} as Response);

		const result = await getHello(fetchImpl as unknown as typeof fetch);

		expect(result).toEqual({ message: 'hi', timestamp: '2026-01-01T00:00:00Z' });
		expect(fetchImpl).toHaveBeenCalledOnce();
	});

	it('throws on non-OK response', async () => {
		const fetchImpl = vi.fn().mockResolvedValue({ ok: false, status: 500 } as Response);

		await expect(getHello(fetchImpl as unknown as typeof fetch)).rejects.toThrow(
			'GET /hello failed: HTTP 500'
		);
	});
});
