<script lang="ts">
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Split } from '$lib/types';
	import { formatTime } from '$lib/format';

	let { splits }: { splits: Split[] } = $props();

	const cumulative = $derived.by(() => {
		let running = 0;
		return splits.map((s) => {
			running += s.timeSeconds;
			return { km: s.km, split: s.timeSeconds, cumulative: running };
		});
	});

	const fastestKm = $derived(
		splits.length === 0
			? null
			: splits.reduce((acc, s) => (s.timeSeconds < acc.timeSeconds ? s : acc))
	);

	const slowestSplit = $derived(
		splits.length === 0 ? 0 : splits.reduce((m, s) => Math.max(m, s.timeSeconds), 0)
	);
</script>

{#if splits.length === 0}
	<p class="opacity-70 text-sm">{i18n.m.splits.none}</p>
{:else}
	<div class="table-wrap">
		<table class="table">
			<thead>
				<tr>
					<th class="w-12">{i18n.m.splits.km}</th>
					<th class="w-24">{i18n.m.splits.split}</th>
					<th>{i18n.m.result.splitsBarTitle}</th>
					<th class="w-24 text-right">{i18n.m.splits.cumulative}</th>
				</tr>
			</thead>
			<tbody>
				{#each cumulative as row (row.km)}
					{@const isFastest = fastestKm?.km === row.km}
					{@const width = slowestSplit > 0 ? (row.split / slowestSplit) * 100 : 0}
					<tr class={isFastest ? 'preset-tonal-success' : ''}>
						<td class="font-mono">{row.km}</td>
						<td class="font-mono">{formatTime(row.split)}</td>
						<td>
							<div class="relative h-2.5 w-full rounded-full bg-surface-200-800 overflow-hidden">
								<div
									class="absolute inset-y-0 left-0 rounded-full {isFastest
										? 'bg-success-500'
										: 'bg-primary-400 dark:bg-primary-500'}"
									style="width: {width}%"
								></div>
							</div>
							{#if isFastest}
								<div class="text-[10px] uppercase tracking-wide text-success-700 dark:text-success-300 mt-0.5">
									{i18n.m.result.fastestKmLabel}
								</div>
							{/if}
						</td>
						<td class="font-mono opacity-80 text-right">{formatTime(row.cumulative)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if fastestKm}
		<p class="text-xs opacity-70 mt-2">
			{i18n.m.splits.fastest(fastestKm.km, formatTime(fastestKm.timeSeconds))}
		</p>
	{/if}
{/if}
