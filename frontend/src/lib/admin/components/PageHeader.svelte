<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		action
	}: { title: string; description?: string; action?: Snippet } = $props();
</script>

<header class="flex items-baseline justify-between gap-4 flex-wrap">
	<div class="space-y-1">
		<h1
			class="text-2xl font-semibold leading-tight"
			style="font-family: var(--heading-font-family);"
		>
			{title}
		</h1>
		{#if description}
			<p class="text-sm opacity-70 max-w-prose">{description}</p>
		{/if}
	</div>
	{#if action}
		<div>{@render action()}</div>
	{/if}
</header>
