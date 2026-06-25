<script lang="ts">
	import { resolve } from '$app/paths';
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import { adminCreateEvent, adminDeleteEvent, adminUpdateEvent } from '$lib/admin/api';
	import type { RaceEvent } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';
	import { formatDate } from '$lib/format';

	type Editing = { mode: 'create' } | { mode: 'edit'; event: RaceEvent } | null;
	let editing: Editing = $state(null);

	let name = $state('');
	let year = $state(new Date().getFullYear());
	let date = $state('');
	let location = $state('');
	let saving = $state(false);
	let errorMsg = $state<string | null>(null);

	function raceCount(eventId: string): number {
		return dataStore.races.filter((r) => r.eventId === eventId).length;
	}

	function openCreate() {
		editing = { mode: 'create' };
		name = `Ingmarsöloppet ${year}`;
		date = '';
		location = 'Stockholm';
		errorMsg = null;
	}

	function openEdit(event: RaceEvent) {
		editing = { mode: 'edit', event };
		name = event.name;
		year = event.year;
		date = event.date;
		location = event.location ?? '';
		errorMsg = null;
	}

	function close() {
		editing = null;
	}

	async function onSave(e: SubmitEvent) {
		e.preventDefault();
		if (!editing) return;
		saving = true;
		errorMsg = null;
		try {
			const payload = {
				name: name.trim(),
				year: Number(year),
				date,
				location: location.trim() || undefined
			};
			if (editing.mode === 'create') {
				await adminCreateEvent(payload);
			} else {
				await adminUpdateEvent({ ...editing.event, ...payload });
			}
			close();
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}

	async function onDelete(event: RaceEvent) {
		if (!confirm(i18n.m.admin.common.confirmDelete)) return;
		await adminDeleteEvent(event.id);
	}
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.events.heading}>
		{#snippet action()}
			<button
				type="button"
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
				onclick={openCreate}
			>
				<Plus class="size-4" />
				{i18n.m.admin.events.add}
			</button>
		{/snippet}
	</PageHeader>

	{#if dataStore.events.length === 0}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.noneYet}</p>
	{:else}
		<div class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden">
			<div class="table-wrap">
				<table class="table">
					<thead>
						<tr>
							<th>{i18n.m.admin.events.columnName}</th>
							<th>{i18n.m.admin.events.columnYear}</th>
							<th>{i18n.m.admin.events.columnDate}</th>
							<th>{i18n.m.admin.events.columnLocation}</th>
							<th>{i18n.m.admin.events.columnRaces}</th>
							<th class="w-40 text-right">{i18n.m.admin.common.actions}</th>
						</tr>
					</thead>
					<tbody>
						{#each dataStore.events as event (event.id)}
							<tr>
								<td class="font-medium">{event.name}</td>
								<td>{event.year}</td>
								<td>{formatDate(event.date)}</td>
								<td>{event.location ?? '—'}</td>
								<td>{raceCount(event.id)}</td>
								<td class="text-right space-x-1">
									<a
										href={resolve('/admin/events/[id]', { id: event.id })}
										class="btn-icon btn-icon-sm hover:bg-surface-100-900 inline-flex"
										aria-label={i18n.m.admin.events.racesHeading}
										title={i18n.m.admin.events.racesHeading}
									>
										<ChevronRight class="size-4" />
									</a>
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-surface-100-900"
										aria-label={i18n.m.admin.common.edit}
										title={i18n.m.admin.common.edit}
										onclick={() => openEdit(event)}
									>
										<Pencil class="size-4" />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
										aria-label={i18n.m.admin.common.delete}
										title={i18n.m.admin.common.delete}
										onclick={() => onDelete(event)}
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
</section>

<Modal
	open={editing !== null}
	title={editing?.mode === 'edit' ? i18n.m.admin.events.edit : i18n.m.admin.events.add}
	onClose={close}
>
	<form class="space-y-4" onsubmit={onSave}>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.events.nameLabel}</span>
			<input type="text" required bind:value={name} class="input" />
		</label>
		<div class="grid grid-cols-2 gap-3">
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.events.yearLabel}</span>
				<input type="number" required min="2000" max="2100" bind:value={year} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.events.dateLabel}</span>
				<input type="date" required bind:value={date} class="input" />
			</label>
		</div>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.events.locationLabel}</span>
			<input type="text" bind:value={location} class="input" />
		</label>

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
