<script lang="ts">
	import Globe from '@lucide/svelte/icons/globe';
	import { i18n } from '$lib/i18n/state.svelte';
	import { locales } from '$lib/i18n/messages';
</script>

<div
	class="inline-flex items-center gap-1 rounded-full border border-surface-200-800 bg-surface-50-950 p-0.5 text-xs"
	role="group"
	aria-label={i18n.m.nav.language}
>
	<Globe class="size-3.5 ml-1.5 opacity-60" aria-hidden="true" />
	{#each locales as l (l.code)}
		<button
			type="button"
			class="rounded-full px-2 py-0.5 transition-colors {i18n.locale === l.code
				? 'bg-primary-500 text-primary-contrast-500 font-semibold'
				: 'opacity-70 hover:opacity-100'}"
			aria-pressed={i18n.locale === l.code}
			title={l.name}
			onclick={() => i18n.set(l.code)}
		>
			{l.code.toUpperCase()}
		</button>
	{/each}
</div>
