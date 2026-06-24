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
</script>

{#if splits.length === 0}
	<p class="opacity-70 text-sm">{i18n.m.splits.none}</p>
{:else}
	<div class="table-wrap">
		<table class="table table-hover">
			<thead>
				<tr>
					<th>{i18n.m.splits.km}</th>
					<th>{i18n.m.splits.split}</th>
					<th>{i18n.m.splits.cumulative}</th>
				</tr>
			</thead>
			<tbody>
				{#each cumulative as row (row.km)}
					<tr class:preset-tonal-success={fastestKm?.km === row.km}>
						<td class="font-mono">{row.km}</td>
						<td class="font-mono">{formatTime(row.split)}</td>
						<td class="font-mono opacity-80">{formatTime(row.cumulative)}</td>
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
