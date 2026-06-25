<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Discipline, ResultExpanded } from '$lib/types';
	import {
		formatCategory,
		formatDate,
		formatDistance,
		formatPace,
		formatRaceName,
		formatTime
	} from '$lib/format';

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

	// Comparator helper: DNF/DNS rows (missing finishSeconds) sort to the
	// end of time/pace orderings regardless of direction. Using Infinity
	// would flip them to the front under desc, which feels wrong — they're
	// "no time" not "very slow".
	function safeTime(r: ResultExpanded): number {
		return r.finishSeconds ?? Number.POSITIVE_INFINITY;
	}

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
					cmp = safeTime(a) - safeTime(b);
					break;
				case 'pace':
					cmp = safeTime(a) / a.race.distanceMeters - safeTime(b) / b.race.distanceMeters;
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
					{@const isFinished = result.status === 'finished' && result.finishSeconds != null}
					<tr
						class="cursor-pointer"
						onclick={() => goto(resolve('/results/[id]', { id: result.id }))}
					>
						<td class="font-mono text-sm">{formatDate(result.event.date)}</td>
						<td>
							<div class="font-medium">{formatRaceName(result.race)}</div>
							<div class="text-xs opacity-70">{result.event.name}</div>
						</td>
						<td>{formatDistance(result.race.distanceMeters)}</td>
						<td class="font-mono">
							{#if isFinished && result.finishSeconds != null}
								{formatTime(result.finishSeconds)}
							{:else}
								<span
									class="inline-block rounded px-2 py-0.5 text-[10px] font-semibold tracking-wide uppercase {result.status ===
									'dns'
										? 'bg-surface-200-800 opacity-80'
										: 'bg-warning-100 text-warning-800 dark:bg-warning-900/40 dark:text-warning-200'}"
								>
									{result.status}
								</span>
							{/if}
						</td>
						<td class="font-mono text-sm">
							{#if isFinished && result.finishSeconds != null}
								{formatPace(result.race.distanceMeters, result.finishSeconds)}
							{:else}
								<span class="opacity-30">—</span>
							{/if}
						</td>
						<td>
							{#if result.placementOverall}
								<span class="font-medium">{result.placementOverall}</span>
								{#if result.placementCategory}
									<span class="text-xs opacity-70">
										({result.placementCategory}
										{formatCategory(result.category)})
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
