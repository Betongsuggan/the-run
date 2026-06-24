<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Race, ResultExpanded } from '$lib/types';
	import { formatDistance, formatPace, formatTime } from '$lib/format';

	let { race, results }: { race: Race; results: ResultExpanded[] } = $props();
</script>

<section class="space-y-2">
	<header class="flex items-baseline justify-between gap-3 flex-wrap">
		<h3 class="h4">{race.name}</h3>
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
						<th class="w-12">{i18n.m.leaderboard.columnRank}</th>
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
							<td class="font-mono">{idx + 1}</td>
							<td>
								<div class="font-medium">{result.runner.name}</div>
							</td>
							<td class="font-mono">{formatTime(result.finishSeconds)}</td>
							<td class="font-mono text-sm">
								{formatPace(race.distanceMeters, result.finishSeconds)}
							</td>
							<td class="text-xs opacity-80">
								{result.category.gender ?? '—'}{#if result.category.ageGroup}
									· {result.category.ageGroup}
								{/if}
							</td>
							<td class="font-mono text-xs opacity-70">{result.bib}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>
