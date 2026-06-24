<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import {
		adminCreateResult,
		adminDeleteResult,
		adminUpdateResult
	} from '$lib/admin/api';
	import type { Gender, Result } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';
	import { formatRaceName, formatTime } from '$lib/format';

	type Editing = { mode: 'create' } | { mode: 'edit'; result: Result } | null;
	let editing: Editing = $state(null);
	let eventFilter = $state<string>('');

	let runnerId = $state('');
	let raceId = $state('');
	let bib = $state('');
	let finishStr = $state('');
	let placementOverallStr = $state('');
	let placementCategoryStr = $state('');
	let categoryGender = $state<Gender | ''>('');
	let categoryAgeGroup = $state('');
	let conditions = $state('');
	let notes = $state('');
	let saving = $state(false);
	let errorMsg = $state<string | null>(null);

	const filteredResults = $derived.by(() => {
		const list = dataStore.results;
		if (!eventFilter) return list;
		const raceIds = new Set(dataStore.races.filter((r) => r.eventId === eventFilter).map((r) => r.id));
		return list.filter((r) => raceIds.has(r.raceId));
	});

	function runnerName(id: string): string {
		return dataStore.runners.find((r) => r.id === id)?.name ?? id;
	}

	function raceLabel(id: string): string {
		const race = dataStore.races.find((r) => r.id === id);
		const event = race && dataStore.events.find((e) => e.id === race.eventId);
		if (!race) return id;
		return event ? `${event.name} — ${formatRaceName(race)}` : formatRaceName(race);
	}

	function eventLabel(raceIdLookup: string): string {
		const race = dataStore.races.find((r) => r.id === raceIdLookup);
		const event = race && dataStore.events.find((e) => e.id === race.eventId);
		return event?.name ?? '—';
	}

	function parseTime(input: string): number | null {
		const m = input.trim().match(/^(?:(\d+):)?(\d{1,2}):(\d{2})$/);
		if (!m) return null;
		const h = m[1] ? Number(m[1]) : 0;
		const mins = Number(m[2]);
		const secs = Number(m[3]);
		if (mins > 59 || secs > 59) return null;
		return h * 3600 + mins * 60 + secs;
	}

	function openCreate() {
		editing = { mode: 'create' };
		runnerId = dataStore.runners[0]?.id ?? '';
		raceId = dataStore.races[0]?.id ?? '';
		bib = '';
		finishStr = '';
		placementOverallStr = '';
		placementCategoryStr = '';
		categoryGender = '';
		categoryAgeGroup = '';
		conditions = '';
		notes = '';
		errorMsg = null;
	}

	function openEdit(result: Result) {
		editing = { mode: 'edit', result };
		runnerId = result.runnerId;
		raceId = result.raceId;
		bib = result.bib;
		finishStr = formatTime(result.finishSeconds);
		placementOverallStr = result.placementOverall ? String(result.placementOverall) : '';
		placementCategoryStr = result.placementCategory ? String(result.placementCategory) : '';
		categoryGender = result.category.gender ?? '';
		categoryAgeGroup = result.category.ageGroup ?? '';
		conditions = result.conditions ?? '';
		notes = result.notes ?? '';
		errorMsg = null;
	}

	function close() {
		editing = null;
	}

	async function onSave(e: SubmitEvent) {
		e.preventDefault();
		if (!editing) return;
		const finishSeconds = parseTime(finishStr);
		if (finishSeconds === null) {
			errorMsg = i18n.m.admin.results.invalidTime;
			return;
		}
		saving = true;
		errorMsg = null;
		try {
			const payload = {
				runnerId,
				raceId,
				bib: bib.trim(),
				finishSeconds,
				placementOverall: placementOverallStr ? Number(placementOverallStr) : undefined,
				placementCategory: placementCategoryStr ? Number(placementCategoryStr) : undefined,
				category: {
					gender: (categoryGender || undefined) as Gender | undefined,
					ageGroup: categoryAgeGroup.trim() || undefined
				},
				conditions: conditions.trim() || undefined,
				notes: notes.trim() || undefined
			};
			if (editing.mode === 'create') {
				await adminCreateResult(payload);
			} else {
				await adminUpdateResult({ ...editing.result, ...payload });
			}
			close();
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}

	async function onDelete(result: Result) {
		if (!confirm(i18n.m.admin.common.confirmDelete)) return;
		await adminDeleteResult(result.id);
	}
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.results.heading}>
		{#snippet action()}
			<button
				type="button"
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
				onclick={openCreate}
				disabled={dataStore.runners.length === 0 || dataStore.races.length === 0}
			>
				<Plus class="size-4" />
				{i18n.m.admin.results.add}
			</button>
		{/snippet}
	</PageHeader>

	<label class="block max-w-sm space-y-1">
		<span class="text-xs uppercase tracking-wide opacity-70">{i18n.m.admin.results.eventFilterLabel}</span>
		<select bind:value={eventFilter} class="select">
			<option value="">{i18n.m.admin.results.eventFilterAll}</option>
			{#each dataStore.events as e (e.id)}
				<option value={e.id}>{e.name}</option>
			{/each}
		</select>
	</label>

	{#if filteredResults.length === 0}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.noneYet}</p>
	{:else}
		<div class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden">
			<div class="table-wrap">
				<table class="table">
					<thead>
						<tr>
							<th>{i18n.m.admin.results.columnEvent}</th>
							<th>{i18n.m.admin.results.columnRace}</th>
							<th>{i18n.m.admin.results.columnRunner}</th>
							<th>{i18n.m.admin.results.columnTime}</th>
							<th>{i18n.m.admin.results.columnPlacement}</th>
							<th class="w-28 text-right">{i18n.m.admin.common.actions}</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredResults as result (result.id)}
							<tr>
								<td class="text-sm opacity-80">{eventLabel(result.raceId)}</td>
								<td>{raceLabel(result.raceId).split(' — ')[1] ?? raceLabel(result.raceId)}</td>
								<td class="font-medium">{runnerName(result.runnerId)}</td>
								<td class="font-mono">{formatTime(result.finishSeconds)}</td>
								<td>{result.placementOverall ?? '—'}</td>
								<td class="text-right space-x-1">
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-surface-100-900"
										aria-label={i18n.m.admin.common.edit}
										title={i18n.m.admin.common.edit}
										onclick={() => openEdit(result)}
									>
										<Pencil class="size-4" />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
										aria-label={i18n.m.admin.common.delete}
										title={i18n.m.admin.common.delete}
										onclick={() => onDelete(result)}
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
	title={editing?.mode === 'edit' ? i18n.m.admin.results.edit : i18n.m.admin.results.add}
	onClose={close}
>
	<form class="space-y-4" onsubmit={onSave}>
		<div class="grid grid-cols-2 gap-3">
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.runnerLabel}</span>
				<select required bind:value={runnerId} class="select">
					{#each dataStore.runners as r (r.id)}
						<option value={r.id}>{r.name}</option>
					{/each}
				</select>
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.raceLabel}</span>
				<select required bind:value={raceId} class="select">
					{#each dataStore.races as r (r.id)}
						<option value={r.id}>{raceLabel(r.id)}</option>
					{/each}
				</select>
			</label>
		</div>

		<div class="grid grid-cols-2 gap-3">
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.bibLabel}</span>
				<input type="text" required bind:value={bib} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.finishTimeLabel}</span>
				<input type="text" required bind:value={finishStr} class="input font-mono" placeholder="24:12" />
			</label>
		</div>

		<div class="grid grid-cols-2 gap-3">
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.placementOverallLabel}</span>
				<input type="number" min="1" bind:value={placementOverallStr} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.placementCategoryLabel}</span>
				<input type="number" min="1" bind:value={placementCategoryStr} class="input" />
			</label>
		</div>

		<div class="grid grid-cols-2 gap-3">
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.categoryGenderLabel}</span>
				<select bind:value={categoryGender} class="select">
					<option value="">—</option>
					<option value="M">M</option>
					<option value="F">F</option>
					<option value="X">X</option>
				</select>
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.results.categoryAgeGroupLabel}</span>
				<input type="text" bind:value={categoryAgeGroup} class="input" placeholder="M30-39" />
			</label>
		</div>

		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.results.conditionsLabel}</span>
			<input type="text" bind:value={conditions} class="input" />
		</label>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.results.notesLabel}</span>
			<textarea bind:value={notes} rows="3" class="textarea"></textarea>
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
