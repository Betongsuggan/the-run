<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Footprints from '@lucide/svelte/icons/footprints';
	import Compass from '@lucide/svelte/icons/compass';
	import Users from '@lucide/svelte/icons/users';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Race, ResultExpanded } from '$lib/types';
	import {
		formatCategory,
		formatDistance,
		formatPace,
		formatRaceName,
		formatTime
	} from '$lib/format';
	import RankBadge from '$lib/components/RankBadge.svelte';

	let { race, results }: { race: Race; results: ResultExpanded[] } = $props();

	const DisciplineIcon = $derived(
		race.discipline === 'run' ? Footprints : race.discipline === 'walk' ? Compass : Users
	);
</script>

<section class="card preset-filled-surface-50-950 border border-surface-200-800 p-5 space-y-3">
	<header class="flex items-baseline justify-between gap-3 flex-wrap">
		<div class="flex items-center gap-2 min-w-0">
			<DisciplineIcon
				class="size-5 text-primary-600 dark:text-primary-300 shrink-0"
				aria-hidden="true"
			/>
			<h3
				class="text-lg font-semibold leading-tight truncate"
				style="font-family: var(--heading-font-family);"
			>
				{formatRaceName(race)}
			</h3>
		</div>
		<div class="text-xs opacity-70">
			{formatDistance(race.distanceMeters)} · {i18n.m.discipline[race.discipline]} · {i18n.m.event.participants(
				results.length
			)}
		</div>
	</header>

	{#if results.length === 0}
		<p class="text-sm opacity-70">{i18n.m.event.noResults}</p>
	{:else}
		<div class="table-wrap">
			<table class="table table-hover">
				<thead>
					<tr>
						<th class="w-16">{i18n.m.leaderboard.columnRank}</th>
						<th>{i18n.m.leaderboard.columnRunner}</th>
						<th>{i18n.m.leaderboard.columnTime}</th>
						<th>{i18n.m.leaderboard.columnPace}</th>
						<th>{i18n.m.leaderboard.columnCategory}</th>
						<th>{i18n.m.leaderboard.columnBib}</th>
					</tr>
				</thead>
				<tbody>
					{#each results as result, idx (result.id)}
						<tr
							class="cursor-pointer"
							onclick={() => goto(resolve('/results/[id]', { id: result.id }))}
						>
							<td>
								{#if idx < 3}
									<RankBadge rank={idx + 1} />
								{:else}
									<span class="font-mono text-sm opacity-70">{idx + 1}</span>
								{/if}
							</td>
							<td>
								<div class="font-medium">{result.runner.name}</div>
							</td>
							<td class="font-mono">{formatTime(result.finishSeconds)}</td>
							<td class="font-mono text-sm">
								{formatPace(race.distanceMeters, result.finishSeconds)}
							</td>
							<td class="text-xs opacity-80">
								{formatCategory(result.category) || '—'}
							</td>
							<td class="font-mono text-xs opacity-70">{result.bib}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>
