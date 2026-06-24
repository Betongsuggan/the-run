<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		children,
		background,
		footer,
		class: className = ''
	}: {
		children: Snippet;
		background?: Snippet;
		footer?: Snippet;
		class?: string;
	} = $props();
</script>

<section
	class="relative overflow-hidden rounded-2xl border border-surface-200-800 bg-gradient-to-b from-primary-50 to-tertiary-50 dark:from-surface-900 dark:to-surface-950 {className}"
>
	{#if background}
		<div class="pointer-events-none absolute inset-0">
			{@render background()}
		</div>
	{/if}
	<div class="relative z-10 px-6 py-10 sm:px-10 sm:py-14">
		{@render children()}
	</div>
	{#if footer}
		<div class="relative z-10">
			{@render footer()}
		</div>
	{/if}
</section>
