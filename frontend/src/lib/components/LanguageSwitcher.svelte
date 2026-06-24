<script lang="ts">
	import { i18n } from '$lib/i18n/state.svelte';
	import { locales } from '$lib/i18n/messages';
</script>

<div class="flex items-center gap-1 text-xs" role="group" aria-label={i18n.m.nav.language}>
	{#each locales as l, i (l.code)}
		{#if i > 0}<span class="opacity-40" aria-hidden="true">·</span>{/if}
		<button
			type="button"
			class="hover:underline px-1"
			class:font-semibold={i18n.locale === l.code}
			class:text-primary-500={i18n.locale === l.code}
			aria-pressed={i18n.locale === l.code}
			title={l.name}
			onclick={() => i18n.set(l.code)}
		>
			{l.code.toUpperCase()}
		</button>
	{/each}
</div>
