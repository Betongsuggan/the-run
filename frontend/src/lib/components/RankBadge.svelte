<script lang="ts">
	import Medal from '@lucide/svelte/icons/medal';

	let { rank }: { rank: number } = $props();

	const tone = $derived(
		rank === 1
			? 'bg-tertiary-200 text-tertiary-950 dark:bg-tertiary-700 dark:text-tertiary-50'
			: rank === 2
				? 'bg-surface-200 text-surface-900 dark:bg-surface-600 dark:text-surface-50'
				: 'bg-warning-200 text-warning-950 dark:bg-warning-800 dark:text-warning-50'
	);
</script>

<span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold {tone}">
	<Medal class="size-3.5" aria-hidden="true" />
	{rank}
</span>
