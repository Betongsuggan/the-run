<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import Calendar from '@lucide/svelte/icons/calendar';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Trophy from '@lucide/svelte/icons/trophy';
	import { getEvent, listRacesForEvent, listResultsForRace } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';
	import { formatDate } from '$lib/format';
	import type { Race, RaceEvent, ResultExpanded } from '$lib/types';
	import RaceLeaderboard from '$lib/components/RaceLeaderboard.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Hero from '$lib/components/Hero.svelte';
	import LighthouseAccent from '$lib/illustrations/LighthouseAccent.svelte';

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
		<a
			href={resolve('/')}
			class="inline-flex items-center gap-1 text-sm opacity-70 hover:opacity-100 transition-opacity"
		>
			<ArrowLeft class="size-4" />
			{i18n.m.event.backHome}
		</a>

		<Hero>
			{#snippet background()}
				<LighthouseAccent
					class="absolute -top-2 right-4 sm:right-8 h-40 w-28 text-secondary-600 dark:text-secondary-400"
				/>
			{/snippet}
			<div class="max-w-xl">
				<h1
					class="text-3xl sm:text-4xl font-semibold leading-tight tracking-tight"
					style="font-family: var(--heading-font-family);"
				>
					{event.name}
				</h1>
				<div class="flex flex-wrap items-center gap-x-5 gap-y-1 text-sm opacity-85 mt-3">
					<span class="flex items-center gap-1.5">
						<Calendar class="size-4 opacity-80" />
						{formatDate(event.date)}
					</span>
					{#if event.location}
						<span class="flex items-center gap-1.5">
							<MapPin class="size-4 opacity-80" />
							{event.location}
						</span>
					{/if}
				</div>
			</div>
		</Hero>

		<section class="space-y-6">
			<SectionHeader title={i18n.m.event.racesHeading} icon={Trophy} tone="primary" />
			{#each boards as board (board.race.id)}
				<RaceLeaderboard race={board.race} results={board.results} />
			{/each}
		</section>
	</section>
{/if}
