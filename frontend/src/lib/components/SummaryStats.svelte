<script lang="ts">
	import { SvelteMap } from 'svelte/reactivity';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded } from '$lib/types';
	import { formatTime, formatDistance } from '$lib/format';

	let { results }: { results: ResultExpanded[] } = $props();

	const totalKm = $derived(
		Math.round((results.reduce((sum, r) => sum + r.race.distanceMeters, 0) / 1000) * 100) / 100
	);
	const raceCount = $derived(results.length);
	const eventCount = $derived(new Set(results.map((r) => r.event.id)).size);

	type Best = { distance: number; result: ResultExpanded };

	const bestsByDistance = $derived.by(() => {
		const map = new SvelteMap<number, Best>();
		for (const r of results) {
			if (r.race.discipline !== 'run') continue;
			const existing = map.get(r.race.distanceMeters);
			if (!existing || r.finishSeconds < existing.result.finishSeconds) {
				map.set(r.race.distanceMeters, { distance: r.race.distanceMeters, result: r });
			}
		}
		return Array.from(map.values()).sort((a, b) => a.distance - b.distance);
	});
</script>

<section class="grid gap-3 sm:grid-cols-3">
	<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
		<div class="text-xs opacity-70 uppercase tracking-wide">{i18n.m.summary.races}</div>
		<div class="text-2xl font-bold">{raceCount}</div>
		<div class="text-xs opacity-70 mt-1">{i18n.m.summary.acrossEvents(eventCount)}</div>
	</div>
	<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
		<div class="text-xs opacity-70 uppercase tracking-wide">{i18n.m.summary.totalDistance}</div>
		<div class="text-2xl font-bold">{totalKm} km</div>
	</div>
	<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
		<div class="text-xs opacity-70 uppercase tracking-wide">{i18n.m.summary.personalBests}</div>
		{#if bestsByDistance.length === 0}
			<div class="text-sm opacity-70 mt-1">—</div>
		{:else}
			<ul class="text-sm mt-1 space-y-0.5">
				{#each bestsByDistance as best (best.distance)}
					<li class="flex justify-between gap-3">
						<span class="opacity-80">{formatDistance(best.distance)}</span>
						<span class="font-mono">{formatTime(best.result.finishSeconds)}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</section>
