<script lang="ts">
	import type { Component, Snippet } from 'svelte';

	let {
		title,
		icon: Icon,
		tone = 'primary',
		action
	}: {
		title: string;
		icon?: Component<{ class?: string }>;
		tone?: 'primary' | 'secondary' | 'tertiary' | 'success';
		action?: Snippet;
	} = $props();

	const accent = {
		primary: 'bg-primary-500',
		secondary: 'bg-secondary-500',
		tertiary: 'bg-tertiary-500',
		success: 'bg-success-500'
	} as const;
</script>

<div class="flex items-center gap-3 mb-4">
	<div class="flex items-center gap-2 min-w-0">
		<span class="inline-block h-6 w-1 rounded-full {accent[tone]}" aria-hidden="true"></span>
		{#if Icon}
			<Icon class="size-5 opacity-70" />
		{/if}
		<h2 class="h3 font-heading truncate" style="font-family: var(--heading-font-family);">
			{title}
		</h2>
	</div>
	{#if action}
		<div class="ml-auto">
			{@render action()}
		</div>
	{/if}
</div>
