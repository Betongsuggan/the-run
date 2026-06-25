<script lang="ts">
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import { formatRaceName } from '$lib/format';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import RegistrationsResultsTable from '$lib/admin/components/RegistrationsResultsTable.svelte';

	const years = $derived(
		Array.from(new Set(dataStore.events.map((e) => e.year))).sort((a, b) => b - a)
	);

	let selectedYear = $state<number | ''>('');
	let selectedRaceId = $state<string>('');
	let tableVersion = $state(0);

	$effect(() => {
		if (selectedYear === '' && years.length > 0) {
			selectedYear = years[0];
		}
	});

	const eventForYear = $derived(
		selectedYear !== '' ? dataStore.events.find((e) => e.year === selectedYear) : undefined
	);

	const racesForYear = $derived(
		eventForYear ? dataStore.races.filter((r) => r.eventId === eventForYear.id) : []
	);

	$effect(() => {
		// Reset race when year changes and current race doesn't belong to the new event.
		if (selectedRaceId && !racesForYear.some((r) => r.id === selectedRaceId)) {
			selectedRaceId = racesForYear[0]?.id ?? '';
		} else if (!selectedRaceId && racesForYear.length > 0) {
			selectedRaceId = racesForYear[0].id;
		}
	});
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.results.heading} />
	<p class="text-sm opacity-80 -mt-2">{i18n.m.admin.results.subheading}</p>

	<div class="flex flex-wrap items-end gap-3">
		<label class="block space-y-1 w-40">
			<span class="text-xs uppercase tracking-wide opacity-70"
				>{i18n.m.admin.results.yearLabel}</span
			>
			<select bind:value={selectedYear} class="select">
				{#each years as y (y)}
					<option value={y}>{y}</option>
				{/each}
			</select>
		</label>
		<label class="block space-y-1 min-w-[18rem] flex-1">
			<span class="text-xs uppercase tracking-wide opacity-70"
				>{i18n.m.admin.results.raceLabel}</span
			>
			<select bind:value={selectedRaceId} class="select" disabled={racesForYear.length === 0}>
				{#each racesForYear as r (r.id)}
					<option value={r.id}>{r.name} — {formatRaceName(r)}</option>
				{/each}
			</select>
		</label>
	</div>

	{#if !selectedRaceId}
		<p class="opacity-70 text-sm">{i18n.m.admin.results.pickPrompt}</p>
	{:else}
		<RegistrationsResultsTable
			raceId={selectedRaceId}
			version={tableVersion}
			onChange={() => (tableVersion += 1)}
		/>
	{/if}
</section>
