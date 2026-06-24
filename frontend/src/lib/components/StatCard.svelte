<script lang="ts">
	import type { Component } from 'svelte';

	let {
		icon: Icon,
		label,
		value,
		sublabel,
		tone = 'primary'
	}: {
		icon: Component<{ class?: string }>;
		label: string;
		value: string | number;
		sublabel?: string;
		tone?: 'primary' | 'secondary' | 'tertiary' | 'success';
	} = $props();

	const toneClasses = {
		primary: 'text-primary-600 dark:text-primary-300 bg-primary-100 dark:bg-primary-900/40',
		secondary: 'text-secondary-600 dark:text-secondary-300 bg-secondary-100 dark:bg-secondary-900/40',
		tertiary: 'text-tertiary-700 dark:text-tertiary-300 bg-tertiary-100 dark:bg-tertiary-900/40',
		success: 'text-success-700 dark:text-success-300 bg-success-100 dark:bg-success-900/40'
	} as const;
</script>

<div
	class="card preset-filled-surface-50-950 border border-surface-200-800 p-4 flex items-start gap-3 transition-shadow hover:shadow-md"
>
	<div
		class="shrink-0 size-10 rounded-xl flex items-center justify-center {toneClasses[tone]}"
		aria-hidden="true"
	>
		<Icon class="size-5" />
	</div>
	<div class="min-w-0">
		<div class="text-xs uppercase tracking-wide opacity-70">{label}</div>
		<div class="text-xl font-semibold leading-tight mt-0.5 truncate">{value}</div>
		{#if sublabel}
			<div class="text-xs opacity-70 mt-0.5">{sublabel}</div>
		{/if}
	</div>
</div>
