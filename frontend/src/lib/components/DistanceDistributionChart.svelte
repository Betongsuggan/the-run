<script lang="ts">
	import { SvelteMap } from 'svelte/reactivity';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Discipline, ResultExpanded } from '$lib/types';
	import { formatDistance } from '$lib/format';

	let { results }: { results: ResultExpanded[] } = $props();

	type Bucket = { key: string; distance: number; discipline: Discipline; count: number };

	const buckets: Bucket[] = $derived.by(() => {
		const map = new SvelteMap<string, Bucket>();
		for (const r of results) {
			const key = `${r.race.distanceMeters}-${r.race.discipline}`;
			const existing = map.get(key);
			if (existing) {
				existing.count += 1;
			} else {
				map.set(key, {
					key,
					distance: r.race.distanceMeters,
					discipline: r.race.discipline,
					count: 1
				});
			}
		}
		return Array.from(map.values()).sort(
			(a, b) => a.distance - b.distance || a.discipline.localeCompare(b.discipline)
		);
	});

	const max = $derived(buckets.reduce((m, b) => Math.max(m, b.count), 0));

	const disciplineClass: Record<Discipline, string> = {
		run: 'fill-primary-500',
		walk: 'fill-secondary-500',
		kids: 'fill-tertiary-500'
	};
</script>

{#if buckets.length === 0}
	<p class="text-sm opacity-70">{i18n.m.chart.noResultsYet}</p>
{:else}
	<div class="space-y-2">
		{#each buckets as b (b.key)}
			<div class="flex items-center gap-3 text-sm">
				<div class="w-20 font-mono opacity-80 text-right">
					{formatDistance(b.distance)}
				</div>
				<div class="flex-1 h-6 bg-surface-200-800 rounded overflow-hidden">
					<svg viewBox="0 0 100 100" preserveAspectRatio="none" class="w-full h-full">
						<rect
							x="0"
							y="0"
							height="100"
							width={(b.count / max) * 100}
							class={disciplineClass[b.discipline]}
						/>
					</svg>
				</div>
				<div class="w-16 text-right">
					<span class="font-bold">{b.count}</span>
					<span class="text-xs opacity-70 ml-1">{i18n.m.discipline[b.discipline]}</span>
				</div>
			</div>
		{/each}
	</div>
{/if}
