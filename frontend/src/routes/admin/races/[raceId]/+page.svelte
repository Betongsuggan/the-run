<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Plus from '@lucide/svelte/icons/plus';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import { adminCreateRegistration } from '$lib/admin/api';
	import { formatDate, formatDistance, formatRaceName } from '$lib/format';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import RegistrationsResultsTable from '$lib/admin/components/RegistrationsResultsTable.svelte';

	const raceId = $derived($page.params.raceId ?? '');
	const race = $derived(dataStore.races.find((r) => r.id === raceId));
	const event = $derived(race ? dataStore.events.find((e) => e.id === race.eventId) : undefined);

	let tableVersion = $state(0);
	let newRunnerId = $state('');
	let newBib = $state('');
	let registering = $state(false);
	let registerError = $state<string | null>(null);

	const registeredRunnerIds = $derived(
		new Set(dataStore.registrations.filter((r) => r.raceId === raceId).map((r) => r.runnerId))
	);

	const availableRunners = $derived(
		dataStore.runners.filter((r) => !registeredRunnerIds.has(r.id))
	);

	$effect(() => {
		if (!newRunnerId && availableRunners.length > 0) {
			newRunnerId = availableRunners[0].id;
		}
	});

	async function onRegister(e: SubmitEvent) {
		e.preventDefault();
		if (!race) return;
		registering = true;
		registerError = null;
		try {
			await adminCreateRegistration({ raceId: race.id, runnerId: newRunnerId, bib: newBib });
			newBib = '';
			newRunnerId = '';
			tableVersion += 1;
		} catch (err) {
			registerError = err instanceof Error ? err.message : String(err);
		} finally {
			registering = false;
		}
	}
</script>

<section class="space-y-6">
	{#if event}
		<a
			href={resolve('/admin/events/[id]', { id: event.id })}
			class="inline-flex items-center gap-1 text-sm opacity-70 hover:opacity-100"
		>
			<ArrowLeft class="size-4" />
			{i18n.m.admin.races.pageBack}
		</a>
	{/if}

	{#if !race}
		<p class="opacity-70">{i18n.m.event.notFound}</p>
	{:else}
		<PageHeader title={race.name} />

		<p class="text-sm opacity-80 -mt-2">
			{event?.name} · {event ? formatDate(event.date) : ''} · {formatRaceName(race)} · {formatDistance(race.distanceMeters)}
		</p>

		<section class="card preset-filled-surface-50-950 border border-surface-200-800 p-4 space-y-3">
			<h2 class="text-sm uppercase tracking-wide opacity-70">{i18n.m.admin.races.registerHeading}</h2>
			{#if availableRunners.length === 0}
				<p class="text-sm opacity-70">{i18n.m.admin.races.noRunnersAvailable}</p>
			{:else}
				<form class="flex flex-wrap items-end gap-3" onsubmit={onRegister}>
					<label class="block space-y-1 min-w-[14rem] flex-1">
						<span class="text-xs uppercase tracking-wide opacity-70">
							{i18n.m.admin.races.runnerSelectLabel}
						</span>
						<select required bind:value={newRunnerId} class="select">
							{#each availableRunners as r (r.id)}
								<option value={r.id}>{r.name}</option>
							{/each}
						</select>
					</label>
					<label class="block space-y-1 w-32">
						<span class="text-xs uppercase tracking-wide opacity-70">{i18n.m.admin.races.bibLabel}</span>
						<input type="text" bind:value={newBib} class="input" />
					</label>
					<button
						type="submit"
						class="btn preset-tonal-primary inline-flex items-center gap-2"
						disabled={registering}
					>
						<Plus class="size-4" />
						{i18n.m.admin.races.registerSubmit}
					</button>
				</form>
				{#if registerError}
					<p class="text-sm text-error-600 dark:text-error-300">{registerError}</p>
				{/if}
			{/if}
		</section>

		<RegistrationsResultsTable
			{raceId}
			version={tableVersion}
			onChange={() => (tableVersion += 1)}
		/>
	{/if}
</section>
