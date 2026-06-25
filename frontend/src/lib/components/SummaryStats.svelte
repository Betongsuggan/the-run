<script lang="ts">
	import { SvelteMap } from 'svelte/reactivity';
	import Trophy from '@lucide/svelte/icons/trophy';
	import Route from '@lucide/svelte/icons/route';
	import Medal from '@lucide/svelte/icons/medal';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded } from '$lib/types';
	import { formatTime, formatDistance } from '$lib/format';
	import StatCard from '$lib/components/StatCard.svelte';

	let { results }: { results: ResultExpanded[] } = $props();

	// Summary stats are about completed races — DNF/DNS rows don't count
	// toward total distance, race count, or personal bests.
	const finished = $derived(
		results.filter(
			(r): r is ResultExpanded & { finishSeconds: number } =>
				r.status === 'finished' && r.finishSeconds != null
		)
	);

	const totalKm = $derived(
		Math.round((finished.reduce((sum, r) => sum + r.race.distanceMeters, 0) / 1000) * 100) / 100
	);
	const raceCount = $derived(finished.length);
	const eventCount = $derived(new Set(finished.map((r) => r.event.id)).size);

	type Best = { distance: number; result: ResultExpanded & { finishSeconds: number } };

	const bestsByDistance = $derived.by(() => {
		const map = new SvelteMap<number, Best>();
		for (const r of finished) {
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
	<StatCard
		icon={Trophy}
		label={i18n.m.summary.races}
		value={raceCount}
		sublabel={i18n.m.summary.acrossEvents(eventCount)}
		tone="primary"
	/>
	<StatCard icon={Route} label={i18n.m.summary.totalDistance} value="{totalKm} km" tone="success" />
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-4 flex gap-3">
		<div
			class="shrink-0 size-10 rounded-xl flex items-center justify-center bg-tertiary-100 text-tertiary-700 dark:bg-tertiary-900/40 dark:text-tertiary-300"
			aria-hidden="true"
		>
			<Medal class="size-5" />
		</div>
		<div class="min-w-0 flex-1">
			<div class="text-xs uppercase tracking-wide opacity-70">{i18n.m.summary.personalBests}</div>
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
	</div>
</section>
