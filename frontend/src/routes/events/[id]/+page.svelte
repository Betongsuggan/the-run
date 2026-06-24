<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { getEvent, listRacesForEvent, listResultsForRace } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';
	import { formatDate } from '$lib/format';
	import type { Race, RaceEvent, ResultExpanded } from '$lib/types';
	import RaceLeaderboard from '$lib/components/RaceLeaderboard.svelte';

	type Board = { race: Race; results: ResultExpanded[] };

	let event: RaceEvent | undefined = $state(undefined);
	let boards: Board[] = $state([]);
	let loading = $state(true);

	$effect(() => {
		const id = $page.params.id;
		if (!id) return;
		loading = true;
		(async () => {
			event = await getEvent(id);
			if (!event) {
				boards = [];
				loading = false;
				return;
			}
			const racesInEvent = await listRacesForEvent(id);
			boards = await Promise.all(
				racesInEvent.map(async (race) => ({
					race,
					results: await listResultsForRace(race.id)
				}))
			);
			loading = false;
		})();
	});
</script>

{#if loading}
	<p class="opacity-70">{i18n.m.event.loading}</p>
{:else if !event}
	<p>
		{i18n.m.event.notFound}
		<a href={resolve('/')} class="text-primary-500 hover:underline">{i18n.m.event.backHome}</a>
	</p>
{:else}
	<section class="space-y-6">
		<div class="text-sm">
			<a href={resolve('/')} class="text-primary-500 hover:underline">← {i18n.m.event.backHome}</a>
		</div>

		<header class="space-y-1">
			<h1 class="h2">{event.name}</h1>
			<div class="opacity-80">
				{formatDate(event.date)}{#if event.location}
					· {event.location}
				{/if}
			</div>
		</header>

		<section class="space-y-8">
			<h2 class="h3">{i18n.m.event.racesHeading}</h2>
			{#each boards as board (board.race.id)}
				<RaceLeaderboard race={board.race} results={board.results} />
			{/each}
		</section>
	</section>
{/if}
