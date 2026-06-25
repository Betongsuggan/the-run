<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	// Cloudflare Turnstile widget. Loads api.js on first mount, renders the
	// widget into a local div, and reports the issued token via the onToken
	// callback. Empty siteKey → render nothing (local dev). Backend mirrors
	// with its own skip flag so the form still submits.
	let { siteKey, onToken }: { siteKey: string; onToken: (token: string) => void } = $props();

	let container: HTMLDivElement | undefined = $state();
	let widgetId: string | undefined;

	const SCRIPT_SRC = 'https://challenges.cloudflare.com/turnstile/v0/api.js';

	function loadScript(): Promise<void> {
		if (typeof window === 'undefined') return Promise.resolve();
		if (window.turnstile) return Promise.resolve();
		const existing = document.querySelector<HTMLScriptElement>(`script[src="${SCRIPT_SRC}"]`);
		if (existing) {
			return new Promise((resolve) => existing.addEventListener('load', () => resolve()));
		}
		return new Promise((resolve, reject) => {
			const s = document.createElement('script');
			s.src = SCRIPT_SRC;
			s.async = true;
			s.defer = true;
			s.onload = () => resolve();
			s.onerror = () => reject(new Error('failed to load Turnstile script'));
			document.head.appendChild(s);
		});
	}

	onMount(async () => {
		if (!siteKey || !container) return;
		await loadScript();
		if (!window.turnstile || !container) return;
		widgetId = window.turnstile.render(container, {
			sitekey: siteKey,
			callback: (token: string) => onToken(token),
			'error-callback': () => onToken(''),
			'expired-callback': () => onToken('')
		});
	});

	onDestroy(() => {
		if (widgetId && window.turnstile) {
			window.turnstile.remove(widgetId);
		}
	});
</script>

{#if siteKey}
	<div bind:this={container} class="cf-turnstile"></div>
{/if}
