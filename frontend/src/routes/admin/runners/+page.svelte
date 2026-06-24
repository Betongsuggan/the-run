<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import {
		adminCreateRunner,
		adminDeleteRunner,
		adminUpdateRunner
	} from '$lib/admin/api';
	import type { Gender, Runner } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';

	type Editing = { mode: 'create' } | { mode: 'edit'; runner: Runner } | null;
	let editing: Editing = $state(null);

	let name = $state('');
	let gender = $state<Gender>('M');
	let birthYearStr = $state('');
	let saving = $state(false);
	let errorMsg = $state<string | null>(null);

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
							<th class="w-32 text-right">{i18n.m.admin.common.actions}</th>
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
