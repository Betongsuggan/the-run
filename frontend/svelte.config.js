import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

// Extract the origin (scheme + host) from PUBLIC_API_BASE_URL so the CSP
// connect-src matches whatever API base URL the bundle is being built for.
// Returns [] when the env var is unset (dev `pnpm build` without an exported
// var) or when the URL is same-origin (covered by 'self'). svelte.config.js
// runs in plain Node before Vite loads .env files, so we read process.env
// directly — the prod build recipe in Justfile exports the var explicitly.
function apiConnectSrc() {
	const raw = process.env.PUBLIC_API_BASE_URL;
	if (!raw) return [];
	try {
		const url = new URL(raw);
		if (url.protocol === 'http:' || url.protocol === 'https:') {
			return [`${url.protocol}//${url.host}`];
		}
	} catch {
		/* unparseable — fall through */
	}
	return [];
}

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),

	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false,
			strict: true
		}),

		// Content Security Policy (GDPR A1.4). SvelteKit emits this as a
		// <meta http-equiv> tag and auto-computes SHA-256 hashes for the
		// inline theme-init script in app.html, so we don't need
		// 'unsafe-inline' for script-src.
		//
		// 'unsafe-inline' for style-src is unavoidable — Tailwind v4 +
		// Skeleton inject dynamic styles via <style> tags whose contents
		// vary per build. The realistic threat model (script injection)
		// is still blocked by the strict script-src.
		//
		// connect-src lists the production API origin explicitly so
		// XHR/fetch from the built site reaches the Lambda. The origin
		// is taken from PUBLIC_API_BASE_URL at build time (the prod
		// recipe in Justfile sets this from Pulumi config). Local dev
		// hits :8080 via the SvelteKit dev server, which doesn't
		// enforce CSP in dev mode.
		//
		// Transport-level headers (HSTS, X-Frame-Options, Referrer-Policy,
		// Permissions-Policy, X-Content-Type-Options) live in the
		// CloudFront ResponseHeadersPolicy — see infra/frontend.
		csp: {
			mode: 'hash',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'unsafe-inline'],
				'font-src': ['self', 'data:'],
				'img-src': ['self', 'data:'],
				'connect-src': ['self', ...apiConnectSrc()],
				'frame-ancestors': ['none'],
				'base-uri': ['self'],
				'form-action': ['self'],
				'object-src': ['none']
			}
		}
	}
};

export default config;
