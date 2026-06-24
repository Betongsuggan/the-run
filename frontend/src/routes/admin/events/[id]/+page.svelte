<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import { adminCreateRace, adminDeleteRace, adminUpdateRace } from '$lib/admin/api';
	import type { Discipline, Race } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';
	import { formatDistance, formatRaceName } from '$lib/format';

	const eventId = $derived($page.params.id ?? '');
	const event = $derived(dataStore.events.find((e) => e.id === eventId));
	const races = $derived(dataStore.races.filter((r) => r.eventId === eventId));

	type Editing = { mode: 'create' } | { mode: 'edit'; race: Race } | null;
	let editing: Editing = $state(null);

	let raceName = $state('');
	let distance = $state(5000);
	let discipline = $state<Discipline>('run');
	let saving = $state(false);
	let errorMsg = $state<string | null>(null);

	function openCreate() {
		editing = { mode: 'create' };
		raceName = '';
		distance = 5000;
		discipline = 'run';
		errorMsg = null;
	}

	function openEdit(race: Race) {
		editing = { mode: 'edit', race };
		raceName = race.name;
		distance = race.distanceMeters;
		discipline = race.discipline;
		errorMsg = null;
	}

	function close() {
		editing = null;
	}

	async function onSave(e: SubmitEvent) {
		e.preventDefault();
		if (!editing || !event) return;
		saving = true;
		errorMsg = null;
		try {
			const payload = {
				eventId: event.id,
				name: raceName.trim(),
				distanceMeters: Number(distance),
				discipline
			};
			if (editing.mode === 'create') {
				await adminCreateRace(payload);
			} else {
				await adminUpdateRace({ ...editing.race, ...payload });
			}
			close();
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}

	async function onDelete(race: Race) {
		if (!confirm(i18n.m.admin.common.confirmDelete)) return;
		await adminDeleteRace(race.id);
	}
</script>

<section class="space-y-6">
	<a
		href={resolve('/admin/events')}
		class="inline-flex items-center gap-1 text-sm opacity-70 hover:opacity-100"
	>
		<ArrowLeft class="size-4" />
		{i18n.m.admin.events.heading}
	</a>

	{#if !event}
		<p class="opacity-70">{i18n.m.event.notFound}</p>
	{:else}
		<PageHeader title={event.name}>
			{#snippet action()}
				<button
					type="button"
					class="btn preset-filled-primary-500 inline-flex items-center gap-2"
					onclick={openCreate}
				>
					<Plus class="size-4" />
					{i18n.m.admin.events.addRace}
				</button>
			{/snippet}
		</PageHeader>

		<h2 class="text-sm uppercase tracking-wide opacity-70">{i18n.m.admin.events.racesHeading}</h2>

		{#if races.length === 0}
			<p class="opacity-70 text-sm">{i18n.m.admin.common.noneYet}</p>
		{:else}
			<div class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden">
				<div class="table-wrap">
					<table class="table">
						<thead>
							<tr>
								<th>{i18n.m.admin.events.raceNameLabel}</th>
								<th>{i18n.m.admin.events.distanceLabel}</th>
								<th>{i18n.m.admin.events.disciplineLabel}</th>
								<th class="w-32 text-right">{i18n.m.admin.common.actions}</th>
							</tr>
						</thead>
						<tbody>
							{#each races as race (race.id)}
								<tr>
									<td>
										<div class="font-medium">{race.name}</div>
										<div class="text-xs opacity-70">{formatRaceName(race)}</div>
									</td>
									<td>{formatDistance(race.distanceMeters)}</td>
									<td>{i18n.m.discipline[race.discipline]}</td>
									<td class="text-right space-x-1">
										<button
											type="button"
											class="btn-icon btn-icon-sm hover:bg-surface-100-900"
											aria-label={i18n.m.admin.common.edit}
											title={i18n.m.admin.common.edit}
											onclick={() => openEdit(race)}
										>
											<Pencil class="size-4" />
										</button>
										<button
											type="button"
											class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
											aria-label={i18n.m.admin.common.delete}
											title={i18n.m.admin.common.delete}
											onclick={() => onDelete(race)}
										>
											<Trash2 class="size-4" />
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	{/if}
</section>

<Modal
	open={editing !== null}
	title={editing?.mode === 'edit' ? i18n.m.admin.events.editRace : i18n.m.admin.events.addRace}
	onClose={close}
>
	<form class="space-y-4" onsubmit={onSave}>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.events.raceNameLabel}</span>
			<input type="text" required bind:value={raceName} class="input" placeholder="5K Run" />
		</label>
		<div class="grid grid-cols-2 gap-3">
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.events.distanceLabel}</span>
				<input type="number" required min="100" step="100" bind:value={distance} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.events.disciplineLabel}</span>
				<select bind:value={discipline} class="select">
					<option value="run">{i18n.m.discipline.run}</option>
					<option value="walk">{i18n.m.discipline.walk}</option>
					<option value="kids">{i18n.m.discipline.kids}</option>
				</select>
			</label>
		</div>

		{#if errorMsg}
			<p class="text-sm text-error-600 dark:text-error-300">{errorMsg}</p>
		{/if}

		<div class="flex items-center justify-end gap-2 pt-2">
			<button type="button" class="btn preset-tonal-surface" onclick={close}>
				{i18n.m.admin.common.cancel}
			</button>
			<button type="submit" disabled={saving} class="btn preset-filled-primary-500">
				{saving ? i18n.m.admin.common.saving : i18n.m.admin.common.save}
			</button>
		</div>
	</form>
</Modal>
