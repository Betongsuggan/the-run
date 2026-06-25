import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

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
		// XHR/fetch from the built site reaches the Lambda. Local dev
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
				'connect-src': ['self', 'https://api.rydback.net'],
				'frame-ancestors': ['none'],
				'base-uri': ['self'],
				'form-action': ['self'],
				'object-src': ['none']
			}
		}
	}
};

export default config;
