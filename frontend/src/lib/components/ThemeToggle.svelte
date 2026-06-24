<script lang="ts">
	import Sun from '@lucide/svelte/icons/sun';
	import Moon from '@lucide/svelte/icons/moon';
	import Monitor from '@lucide/svelte/icons/monitor';
	import { theme } from '$lib/theme/state.svelte';
	import { i18n } from '$lib/i18n/state.svelte';

	const label = $derived(
		theme.mode === 'light'
			? i18n.m.nav.themeLight
			: theme.mode === 'dark'
				? i18n.m.nav.themeDark
				: i18n.m.nav.themeSystem
	);
</script>

<button
	type="button"
	class="btn-icon btn-icon-sm preset-tonal-surface hover:preset-filled-primary-100-900 transition-colors"
	aria-label="{i18n.m.nav.theme}: {label}"
	title="{i18n.m.nav.theme}: {label}"
	onclick={() => theme.cycle()}
>
	{#if theme.mode === 'light'}
		<Sun class="size-4" />
	{:else if theme.mode === 'dark'}
		<Moon class="size-4" />
	{:else}
		<Monitor class="size-4" />
	{/if}
</button>
