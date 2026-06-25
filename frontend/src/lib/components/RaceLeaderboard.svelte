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
					{#each results as result (result.id)}
						{@const isFinished = result.status === 'finished' && result.finishSeconds != null}
						<tr
							class="cursor-pointer"
							onclick={() => goto(resolve('/results/[id]', { id: result.id }))}
						>
							<td>
								{#if isFinished && result.placementOverall != null && result.placementOverall <= 3}
									<RankBadge rank={result.placementOverall} />
								{:else if isFinished && result.placementOverall != null}
									<span class="font-mono text-sm opacity-70">{result.placementOverall}</span>
								{:else}
									<span class="font-mono text-sm opacity-30">—</span>
								{/if}
							</td>
							<td>
								<div class="font-medium">{result.runner.name}</div>
							</td>
							<td class="font-mono">
								{#if isFinished && result.finishSeconds != null}
									{formatTime(result.finishSeconds)}
								{:else}
									<!-- Status badge replaces the time/pace for non-finished
									     rows. DNF is public; DNS is admin/owner-only and
									     filtered server-side. -->
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
									{formatPace(race.distanceMeters, result.finishSeconds)}
								{:else}
									<span class="opacity-30">—</span>
								{/if}
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
