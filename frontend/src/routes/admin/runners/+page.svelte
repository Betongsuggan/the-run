<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import ClipboardPlus from '@lucide/svelte/icons/clipboard-plus';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import {
		adminCreateRegistration,
		adminCreateRunner,
		adminDeleteRunner,
		adminUpdateRunner
	} from '$lib/admin/api';
	import type { Gender, Runner } from '$lib/types';
	import { formatRaceName } from '$lib/format';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';

	type Editing = { mode: 'create' } | { mode: 'edit'; runner: Runner } | null;
	let editing: Editing = $state(null);

	let name = $state('');
	let gender = $state<Gender>('M');
	let birthYearStr = $state('');
	let saving = $state(false);
	let errorMsg = $state<string | null>(null);

	// Register-for-race modal state
	let registerFor: Runner | null = $state(null);
	let registerRaceId = $state('');
	let registerBib = $state('');
	let registering = $state(false);
	let registerError = $state<string | null>(null);

	function openCreate() {
		editing = { mode: 'create' };
		name = '';
		gender = 'M';
		birthYearStr = '';
		errorMsg = null;
	}

	function openEdit(runner: Runner) {
		editing = { mode: 'edit', runner };
		name = runner.name;
		gender = runner.gender;
		birthYearStr = runner.birthYear ? String(runner.birthYear) : '';
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
			const birthYear = birthYearStr ? Number(birthYearStr) : undefined;
			if (editing.mode === 'create') {
				await adminCreateRunner({ name: name.trim(), gender, birthYear });
			} else {
				await adminUpdateRunner({ ...editing.runner, name: name.trim(), gender, birthYear });
			}
			close();
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}

	async function onDelete(runner: Runner) {
		if (!confirm(i18n.m.admin.common.confirmDelete)) return;
		await adminDeleteRunner(runner.id);
	}

	function openRegister(runner: Runner) {
		registerFor = runner;
		registerBib = '';
		registerError = null;
		const taken = new Set(
			dataStore.registrations.filter((r) => r.runnerId === runner.id).map((r) => r.raceId)
		);
		const first = dataStore.races.find((r) => !taken.has(r.id));
		registerRaceId = first?.id ?? '';
	}

	function closeRegister() {
		registerFor = null;
	}

	const availableRacesForRegister = $derived.by(() => {
		if (!registerFor) return [];
		const taken = new Set(
			dataStore.registrations
				.filter((r) => r.runnerId === registerFor!.id)
				.map((r) => r.raceId)
		);
		return dataStore.races.filter((r) => !taken.has(r.id));
	});

	function eventNameForRace(raceId: string): string {
		const race = dataStore.races.find((r) => r.id === raceId);
		if (!race) return '';
		return dataStore.events.find((e) => e.id === race.eventId)?.name ?? '';
	}

	async function onRegisterSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!registerFor || !registerRaceId) return;
		registering = true;
		registerError = null;
		try {
			await adminCreateRegistration({
				raceId: registerRaceId,
				runnerId: registerFor.id,
				bib: registerBib
			});
			closeRegister();
		} catch (err) {
			registerError = err instanceof Error ? err.message : String(err);
		} finally {
			registering = false;
		}
	}
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.runners.heading}>
		{#snippet action()}
			<button type="button" class="btn preset-filled-primary-500 inline-flex items-center gap-2" onclick={openCreate}>
				<Plus class="size-4" />
				{i18n.m.admin.runners.add}
			</button>
		{/snippet}
	</PageHeader>

	{#if dataStore.runners.length === 0}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.noneYet}</p>
	{:else}
		<div class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden">
			<div class="table-wrap">
				<table class="table">
					<thead>
						<tr>
							<th>{i18n.m.admin.runners.columnName}</th>
							<th>{i18n.m.admin.runners.columnGender}</th>
							<th>{i18n.m.admin.runners.columnBirthYear}</th>
							<th class="w-40 text-right">{i18n.m.admin.common.actions}</th>
						</tr>
					</thead>
					<tbody>
						{#each dataStore.runners as runner (runner.id)}
							<tr>
								<td class="font-medium">{runner.name}</td>
								<td>{runner.gender}</td>
								<td>{runner.birthYear ?? '—'}</td>
								<td class="text-right space-x-1">
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-900/40 dark:hover:text-primary-200"
										aria-label={i18n.m.admin.runners.registerForRace}
										title={i18n.m.admin.runners.registerForRace}
										onclick={() => openRegister(runner)}
									>
										<ClipboardPlus class="size-4" />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-surface-100-900"
										aria-label={i18n.m.admin.common.edit}
										title={i18n.m.admin.common.edit}
										onclick={() => openEdit(runner)}
									>
										<Pencil class="size-4" />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
										aria-label={i18n.m.admin.common.delete}
										title={i18n.m.admin.common.delete}
										onclick={() => onDelete(runner)}
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
	title={editing?.mode === 'edit' ? i18n.m.admin.runners.edit : i18n.m.admin.runners.add}
	onClose={close}
>
	<form class="space-y-4" onsubmit={onSave}>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.runners.nameLabel}</span>
			<input type="text" required bind:value={name} class="input" />
		</label>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.runners.genderLabel}</span>
			<select bind:value={gender} class="select">
				<option value="M">M</option>
				<option value="F">F</option>
				<option value="X">X</option>
			</select>
		</label>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.runners.birthYearLabel}</span>
			<input
				type="number"
				min="1900"
				max="2100"
				bind:value={birthYearStr}
				class="input"
				placeholder="e.g. 1988"
			/>
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

<Modal
	open={registerFor !== null}
	title={registerFor
		? i18n.m.admin.runners.registerHeading(registerFor.name)
		: i18n.m.admin.runners.registerForRace}
	onClose={closeRegister}
>
	{#if availableRacesForRegister.length === 0}
		<p class="text-sm opacity-70">{i18n.m.admin.runners.noRacesAvailable}</p>
		<div class="flex items-center justify-end pt-4">
			<button type="button" class="btn preset-tonal-surface" onclick={closeRegister}>
				{i18n.m.admin.common.cancel}
			</button>
		</div>
	{:else}
		<form class="space-y-4" onsubmit={onRegisterSubmit}>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.runners.raceSelectLabel}</span>
				<select required bind:value={registerRaceId} class="select">
					{#each availableRacesForRegister as r (r.id)}
						<option value={r.id}>{eventNameForRace(r.id)} — {r.name} ({formatRaceName(r)})</option>
					{/each}
				</select>
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.races.bibLabel}</span>
				<input type="text" bind:value={registerBib} class="input" />
			</label>

			{#if registerError}
				<p class="text-sm text-error-600 dark:text-error-300">{registerError}</p>
			{/if}

			<div class="flex items-center justify-end gap-2 pt-2">
				<button type="button" class="btn preset-tonal-surface" onclick={closeRegister}>
					{i18n.m.admin.common.cancel}
				</button>
				<button type="submit" disabled={registering} class="btn preset-filled-primary-500">
					{registering ? i18n.m.admin.common.saving : i18n.m.admin.races.registerSubmit}
				</button>
			</div>
		</form>
	{/if}
</Modal>
