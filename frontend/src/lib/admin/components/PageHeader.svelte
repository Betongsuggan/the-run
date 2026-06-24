<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, action }: { title: string; action?: Snippet } = $props();
</script>

<header class="flex items-center justify-between gap-4 flex-wrap">
	<h1
		class="text-2xl font-semibold leading-tight"
		style="font-family: var(--heading-font-family);"
	>
		{title}
	</h1>
	{#if action}
		<div>{@render action()}</div>
	{/if}
</header>
