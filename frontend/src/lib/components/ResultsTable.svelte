<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Discipline, ResultExpanded } from '$lib/types';
	import { formatDate, formatDistance, formatPace, formatTime } from '$lib/format';

	let { results }: { results: ResultExpanded[] } = $props();

	type SortKey = 'date' | 'distance' | 'time' | 'pace' | 'placement';
	let sortKey: SortKey = $state('date');
	let sortDir: 'asc' | 'desc' = $state('desc');
	let disciplineFilter: Discipline | 'all' = $state('all');

	const disciplines: Array<Discipline | 'all'> = ['all', 'run', 'walk', 'kids'];

	function disciplineLabel(d: Discipline | 'all'): string {
		return i18n.m.discipline[d];
	}

	const filtered = $derived(
		disciplineFilter === 'all'
			? results
			: results.filter((r) => r.race.discipline === disciplineFilter)
	);

	const sorted = $derived.by(() => {
		const out = [...filtered];
		out.sort((a, b) => {
			let cmp = 0;
			switch (sortKey) {
				case 'date':
					cmp = a.event.date < b.event.date ? -1 : a.event.date > b.event.date ? 1 : 0;
					break;
				case 'distance':
					cmp = a.race.distanceMeters - b.race.distanceMeters;
					break;
				case 'time':
					cmp = a.finishSeconds - b.finishSeconds;
					break;
				case 'pace':
					cmp = a.finishSeconds / a.race.distanceMeters - b.finishSeconds / b.race.distanceMeters;
					break;
				case 'placement':
					cmp = (a.placementOverall ?? Infinity) - (b.placementOverall ?? Infinity);
					break;
			}
			return sortDir === 'asc' ? cmp : -cmp;
		});
		return out;
	});

	function toggleSort(key: SortKey) {
		if (sortKey === key) {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		} else {
			sortKey = key;
			sortDir = key === 'date' ? 'desc' : 'asc';
		}
	}

	function arrow(key: SortKey): string {
		if (sortKey !== key) return '';
		return sortDir === 'asc' ? ' ▲' : ' ▼';
	}
</script>

<div class="space-y-3">
	<div class="flex items-center gap-2 flex-wrap">
		<span class="text-xs opacity-70 uppercase tracking-wide">{i18n.m.resultsTable.filter}</span>
		{#each disciplines as d (d)}
			<button
				type="button"
				class="btn btn-sm"
				class:preset-filled-primary-500={disciplineFilter === d}
				class:preset-tonal={disciplineFilter !== d}
				onclick={() => (disciplineFilter = d)}
			>
				{disciplineLabel(d)}
			</button>
		{/each}
	</div>

	<div class="table-wrap">
		<table class="table table-hover">
			<thead>
				<tr>
					<th>
						<button type="button" class="text-left" onclick={() => toggleSort('date')}
							>{i18n.m.resultsTable.columnDate}{arrow('date')}</button
						>
					</th>
					<th>{i18n.m.resultsTable.columnEventRace}</th>
					<th>
						<button type="button" class="text-left" onclick={() => toggleSort('distance')}
							>{i18n.m.resultsTable.columnDistance}{arrow('distance')}</button
						>
					</th>
					<th>
						<button type="button" class="text-left" onclick={() => toggleSort('time')}
							>{i18n.m.resultsTable.columnTime}{arrow('time')}</button
						>
					</th>
					<th>
						<button type="button" class="text-left" onclick={() => toggleSort('pace')}
							>{i18n.m.resultsTable.columnPace}{arrow('pace')}</button
						>
					</th>
					<th>
						<button type="button" class="text-left" onclick={() => toggleSort('placement')}
							>{i18n.m.resultsTable.columnPlace}{arrow('placement')}</button
						>
					</th>
					<th>{i18n.m.resultsTable.columnBib}</th>
				</tr>
			</thead>
			<tbody>
				{#each sorted as result (result.id)}
					<tr
						class="cursor-pointer"
						onclick={() => goto(resolve('/results/[id]', { id: result.id }))}
					>
						<td class="font-mono text-sm">{formatDate(result.event.date)}</td>
						<td>
							<div class="font-medium">{result.race.name}</div>
							<div class="text-xs opacity-70">{result.event.name}</div>
						</td>
						<td>{formatDistance(result.race.distanceMeters)}</td>
						<td class="font-mono">{formatTime(result.finishSeconds)}</td>
						<td class="font-mono text-sm">
							{formatPace(result.race.distanceMeters, result.finishSeconds)}
						</td>
						<td>
							{#if result.placementOverall}
								<span class="font-medium">{result.placementOverall}</span>
								{#if result.placementCategory}
									<span class="text-xs opacity-70">
										({result.placementCategory}
										{result.category.ageGroup ?? ''})
									</span>
								{/if}
							{:else}
								—
							{/if}
						</td>
						<td class="text-xs opacity-70 font-mono">{result.bib}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if sorted.length === 0}
		<p class="opacity-70 text-sm">{i18n.m.resultsTable.noMatch}</p>
	{/if}
</div>
