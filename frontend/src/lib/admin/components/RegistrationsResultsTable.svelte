<script lang="ts">
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Save from '@lucide/svelte/icons/save';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore, type RegistrationUpdate } from '$lib/store/data.svelte';
	import {
		adminBulkUpdateRegistrations,
		adminDeleteRegistration,
		adminListRegistrationsForRace
	} from '$lib/admin/api';
	import type { Registration, RegistrationStatus } from '$lib/types';
	import { formatCategory, formatTime, parseFinishTime } from '$lib/format';

	let {
		raceId,
		version = 0,
		onChange
	}: { raceId: string; version?: number; onChange?: () => void } = $props();

	type Row = {
		id: string;
		runnerId: string;
		bib: string;
		category: Registration['category'];
		status: RegistrationStatus;
		finishStr: string;
		paid: boolean;
		paymentReceivedAt: string | null;
		original: { bib: string; status: RegistrationStatus; finishStr: string; paid: boolean };
		rowError: string | null;
	};

	let rows: Row[] = $state([]);
	let loading = $state(true);
	let saving = $state(false);
	let saveError = $state<string | null>(null);
	let savedAt = $state<number | null>(null);
	let activeRaceId = $state('');
	let activeVersion = $state(-1);

	function toRow(reg: Registration): Row {
		const finishStr = reg.finishSeconds !== undefined ? formatTime(reg.finishSeconds) : '';
		const bib = reg.bib ?? '';
		const paid = !!reg.paymentReceivedAt;
		return {
			id: reg.id,
			runnerId: reg.runnerId,
			bib,
			category: reg.category,
			status: reg.status,
			finishStr,
			paid,
			paymentReceivedAt: reg.paymentReceivedAt ?? null,
			original: { bib, status: reg.status, finishStr, paid },
			rowError: null
		};
	}

	async function refresh() {
		if (!raceId) {
			rows = [];
			loading = false;
			return;
		}
		loading = true;
		const list = await adminListRegistrationsForRace(raceId);
		list.sort((a, b) => {
			const nameA = dataStore.runners.find((r) => r.id === a.runnerId)?.name ?? '';
			const nameB = dataStore.runners.find((r) => r.id === b.runnerId)?.name ?? '';
			return nameA.localeCompare(nameB);
		});
		rows = list.map(toRow);
		loading = false;
	}

	$effect(() => {
		if (raceId !== activeRaceId || version !== activeVersion) {
			activeRaceId = raceId;
			activeVersion = version;
			saveError = null;
			savedAt = null;
			void refresh();
		}
	});

	function runnerName(id: string): string {
		return dataStore.runners.find((r) => r.id === id)?.name ?? id;
	}

	function isDirty(row: Row): boolean {
		return (
			row.bib !== row.original.bib ||
			row.status !== row.original.status ||
			row.finishStr.trim() !== row.original.finishStr.trim() ||
			row.paid !== row.original.paid
		);
	}

	const dirtyCount = $derived(rows.filter(isDirty).length);

	const race = $derived(dataStore.races.find((r) => r.id === raceId));
	const hasFee = $derived((race?.registrationFeeOre ?? 0) > 0);
	const paidCount = $derived(rows.filter((r) => r.paid).length);

	async function onRemove(row: Row) {
		if (!confirm(i18n.m.admin.races.confirmRemoveRegistration)) return;
		await adminDeleteRegistration(raceId, row.runnerId);
		await refresh();
		onChange?.();
	}

	async function onSave() {
		saveError = null;
		const updates: RegistrationUpdate[] = [];
		let anyRowError = false;
		rows = rows.map((row) => ({ ...row, rowError: null }));

		for (const row of rows) {
			if (!isDirty(row)) continue;
			let finishSeconds: number | null;
			if (row.status === 'finished') {
				const parsed = parseFinishTime(row.finishStr);
				if (parsed === null) {
					row.rowError =
						row.finishStr.trim() === ''
							? i18n.m.admin.races.mustHaveFinishTime
							: i18n.m.admin.races.invalidTime;
					anyRowError = true;
					continue;
				}
				finishSeconds = parsed;
			} else {
				finishSeconds = null;
			}
			const update: RegistrationUpdate = {
				raceId,
				runnerId: row.runnerId,
				status: row.status,
				finishSeconds,
				bib: row.bib
			};
			if (row.paid !== row.original.paid) {
				update.paymentReceivedAt = row.paid ? 'now' : '';
			}
			updates.push(update);
		}

		if (anyRowError) {
			rows = [...rows];
			return;
		}
		if (updates.length === 0) return;

		saving = true;
		try {
			await adminBulkUpdateRegistrations(updates);
			await refresh();
			savedAt = Date.now();
			onChange?.();
		} catch (err) {
			saveError = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}
</script>

{#if loading}
	<p class="opacity-70 text-sm">{i18n.m.admin.common.loading}</p>
{:else if rows.length === 0}
	<p class="opacity-70 text-sm">{i18n.m.admin.races.emptyState}</p>
{:else}
	<div class="space-y-3">
		<div class="flex items-center justify-between gap-3">
			<div class="space-y-0.5">
				<h2 class="text-sm uppercase tracking-wide opacity-70">
					{i18n.m.admin.races.registeredHeading(rows.length)}
				</h2>
				{#if hasFee}
					<p class="text-xs opacity-70">
						{i18n.m.admin.races.paidSummary({ paid: paidCount, total: rows.length })}
					</p>
				{/if}
			</div>
			<button
				type="button"
				class="btn preset-filled-primary-500 inline-flex items-center gap-2 disabled:opacity-50"
				onclick={onSave}
				disabled={saving || dirtyCount === 0}
			>
				<Save class="size-4" />
				{dirtyCount > 0
					? i18n.m.admin.races.saveChangesWithCount(dirtyCount)
					: i18n.m.admin.races.saveChanges}
			</button>
		</div>

		{#if saveError}
			<p class="text-sm text-error-600 dark:text-error-300">{saveError}</p>
		{:else if savedAt !== null && dirtyCount === 0}
			<p class="text-sm text-success-700 dark:text-success-300">{i18n.m.admin.races.saveSuccess}</p>
		{/if}

		<div class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden">
			<div class="table-wrap">
				<table class="table">
					<thead>
						<tr>
							<th>{i18n.m.admin.races.columnRunner}</th>
							{#if hasFee}
								<th class="w-20">{i18n.m.admin.races.columnPaid}</th>
							{/if}
							<th class="w-28">{i18n.m.admin.races.columnBib}</th>
							<th class="w-32">{i18n.m.admin.races.columnCategory}</th>
							<th class="w-36">{i18n.m.admin.races.columnStatus}</th>
							<th class="w-44">{i18n.m.admin.races.columnTime}</th>
							<th class="w-16"></th>
						</tr>
					</thead>
					<tbody>
						{#each rows as row, idx (row.id)}
							{@const dirty = isDirty(row)}
							<tr class={dirty ? 'preset-tonal-warning/30' : ''}>
								<td class="font-medium">{runnerName(row.runnerId)}</td>
								{#if hasFee}
									<td class="text-center">
										<input
											type="checkbox"
											class="checkbox"
											bind:checked={rows[idx].paid}
											aria-label={i18n.m.admin.races.columnPaid}
										/>
									</td>
								{/if}
								<td>
									<input type="text" bind:value={rows[idx].bib} class="input input-sm" />
								</td>
								<td class="text-sm opacity-80">
									{formatCategory(row.category ?? {}) || '—'}
								</td>
								<td>
									<select bind:value={rows[idx].status} class="select select-sm">
										<option value="pending">{i18n.m.admin.races.status.pending}</option>
										<option value="finished">{i18n.m.admin.races.status.finished}</option>
										<option value="dnf">{i18n.m.admin.races.status.dnf}</option>
										<option value="dns">{i18n.m.admin.races.status.dns}</option>
									</select>
								</td>
								<td>
									<input
										type="text"
										bind:value={rows[idx].finishStr}
										class="input input-sm font-mono"
										placeholder="mm:ss"
										disabled={row.status === 'dnf' || row.status === 'dns'}
									/>
									{#if row.rowError}
										<div class="text-xs text-error-600 dark:text-error-300 mt-1">
											{row.rowError}
										</div>
									{/if}
								</td>
								<td class="text-right">
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
										aria-label={i18n.m.admin.common.delete}
										title={i18n.m.admin.common.delete}
										onclick={() => onRemove(row)}
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
	</div>
{/if}
