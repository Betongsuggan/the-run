<script lang="ts">
	import { onMount } from 'svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Eye from '@lucide/svelte/icons/eye';
	import Send from '@lucide/svelte/icons/send';
	import Archive from '@lucide/svelte/icons/archive';
	import History from '@lucide/svelte/icons/history';
	import { i18n } from '$lib/i18n/state.svelte';
	import {
		adminArchivePolicy,
		adminCreatePolicy,
		adminEditPolicy,
		adminListPolicies,
		adminListPolicyRevisions,
		adminPublishPolicy
	} from '$lib/admin/api';
	import type { Policy, PolicyRevision } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';
	import { renderPolicyMarkdown } from '$lib/policy/render';

	type EditingMode =
		| { kind: 'create' }
		| { kind: 'edit'; policy: Policy }
		| { kind: 'history'; policy: Policy; revisions: PolicyRevision[] }
		| null;

	let policies = $state<Policy[]>([]);
	let loading = $state(true);
	let listError = $state<string | null>(null);

	let editing = $state<EditingMode>(null);

	// Form state — reused by both Create and Edit flows.
	let slug = $state('');
	let policyKind = $state<'privacy'>('privacy');
	let effectiveFrom = $state('');
	let bodySv = $state('');
	let bodyEn = $state('');
	let note = $state('');
	let saving = $state(false);
	let formError = $state<string | null>(null);
	let activeTab = $state<'sv' | 'en' | 'preview'>('sv');

	onMount(() => {
		void reload();
	});

	async function reload() {
		loading = true;
		listError = null;
		try {
			policies = await adminListPolicies();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	function openCreate() {
		editing = { kind: 'create' };
		slug = todayDateSlug();
		policyKind = 'privacy';
		effectiveFrom = new Date().toISOString();
		bodySv = '';
		bodyEn = '';
		note = '';
		formError = null;
		activeTab = 'sv';
	}

	function openEdit(policy: Policy) {
		editing = { kind: 'edit', policy };
		slug = policy.slug;
		policyKind = policy.kind;
		effectiveFrom = policy.effectiveFrom;
		bodySv = policy.bodySv;
		bodyEn = policy.bodyEn;
		note = '';
		formError = null;
		activeTab = 'sv';
	}

	async function openHistory(policy: Policy) {
		try {
			const revisions = await adminListPolicyRevisions(policy.id);
			editing = { kind: 'history', policy, revisions };
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		}
	}

	function close() {
		editing = null;
	}

	async function onSave(ev: SubmitEvent) {
		ev.preventDefault();
		if (!editing || editing.kind === 'history') return;
		saving = true;
		formError = null;
		try {
			if (editing.kind === 'create') {
				await adminCreatePolicy({
					kind: policyKind,
					slug: slug.trim(),
					effectiveFrom,
					bodySv,
					bodyEn
				});
			} else {
				await adminEditPolicy(editing.policy.id, {
					bodySv,
					bodyEn,
					note: note.trim() || undefined
				});
			}
			close();
			await reload();
		} catch (err) {
			formError = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}

	async function onPublish(policy: Policy) {
		if (!confirm(i18n.m.admin.policies.confirmPublish(policy.slug))) return;
		try {
			await adminPublishPolicy(policy.id);
			await reload();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		}
	}

	async function onArchive(policy: Policy) {
		if (!confirm(i18n.m.admin.policies.confirmArchive(policy.slug))) return;
		try {
			await adminArchivePolicy(policy.id);
			await reload();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		}
	}

	function todayDateSlug(): string {
		const d = new Date();
		const y = d.getUTCFullYear();
		const m = String(d.getUTCMonth() + 1).padStart(2, '0');
		const day = String(d.getUTCDate()).padStart(2, '0');
		return `${y}-${m}-${day}`;
	}

	const drafts = $derived(policies.filter((p) => p.status === 'draft'));
	const published = $derived(policies.filter((p) => p.status === 'published'));
	const archived = $derived(policies.filter((p) => p.status === 'archived'));

	const previewHtml = $derived(
		activeTab === 'preview'
			? renderPolicyMarkdown((bodySv || '').trim() ? bodySv : bodyEn)
			: ''
	);

	function statusBadgeClass(status: Policy['status']): string {
		switch (status) {
			case 'published':
				return 'badge preset-filled-success-500';
			case 'draft':
				return 'badge preset-filled-warning-500';
			case 'archived':
				return 'badge preset-tonal-surface';
			default:
				return 'badge preset-tonal-surface';
		}
	}

	function fmt(ts: string | undefined): string {
		if (!ts) return '—';
		try {
			return new Date(ts).toLocaleString();
		} catch {
			return ts;
		}
	}
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.policies.heading} description={i18n.m.admin.policies.description}>
		{#snippet action()}
			<button
				type="button"
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
				onclick={openCreate}
			>
				<Plus class="size-4" />
				{i18n.m.admin.policies.add}
			</button>
		{/snippet}
	</PageHeader>

	{#if listError}
		<p class="text-sm text-error-600 dark:text-error-300">{listError}</p>
	{/if}

	{#if loading}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.loading}</p>
	{:else}
		{#each [{ label: i18n.m.admin.policies.sectionPublished, list: published }, { label: i18n.m.admin.policies.sectionDrafts, list: drafts }, { label: i18n.m.admin.policies.sectionArchived, list: archived }] as group (group.label)}
			{#if group.list.length > 0}
				<div class="space-y-2">
					<h2 class="text-sm font-semibold uppercase tracking-wide opacity-70">{group.label}</h2>
					<div
						class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden"
					>
						<div class="table-wrap">
							<table class="table">
								<thead>
									<tr>
										<th>{i18n.m.admin.policies.columnKind}</th>
										<th>{i18n.m.admin.policies.columnSlug}</th>
										<th>{i18n.m.admin.policies.columnStatus}</th>
										<th>{i18n.m.admin.policies.columnRevision}</th>
										<th>{i18n.m.admin.policies.columnEffectiveFrom}</th>
										<th>{i18n.m.admin.policies.columnUpdated}</th>
										<th class="w-56 text-right">{i18n.m.admin.common.actions}</th>
									</tr>
								</thead>
								<tbody>
									{#each group.list as policy (policy.id)}
										<tr>
											<td class="text-sm">{i18n.m.policies.kindLabel[policy.kind] ?? policy.kind}</td>
											<td class="font-mono text-sm">{policy.slug}</td>
											<td>
												<span class={statusBadgeClass(policy.status)}
													>{i18n.m.admin.policies.statusLabel(policy.status)}</span
												>
											</td>
											<td>{policy.revision}</td>
											<td>{fmt(policy.effectiveFrom)}</td>
											<td>{fmt(policy.updatedAt)}</td>
											<td class="text-right space-x-1">
												<button
													type="button"
													class="btn-icon btn-icon-sm hover:bg-surface-100-900"
													aria-label={i18n.m.admin.policies.history}
													title={i18n.m.admin.policies.history}
													onclick={() => openHistory(policy)}
												>
													<History class="size-4" />
												</button>
												{#if policy.status !== 'archived'}
													<button
														type="button"
														class="btn-icon btn-icon-sm hover:bg-surface-100-900"
														aria-label={i18n.m.admin.common.edit}
														title={i18n.m.admin.common.edit}
														onclick={() => openEdit(policy)}
													>
														<Pencil class="size-4" />
													</button>
												{/if}
												{#if policy.status === 'draft'}
													<button
														type="button"
														class="btn-icon btn-icon-sm hover:bg-success-100 hover:text-success-700 dark:hover:bg-success-900/40 dark:hover:text-success-200"
														aria-label={i18n.m.admin.policies.publish}
														title={i18n.m.admin.policies.publish}
														onclick={() => onPublish(policy)}
													>
														<Send class="size-4" />
													</button>
												{/if}
												{#if policy.status === 'published'}
													<button
														type="button"
														class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
														aria-label={i18n.m.admin.policies.archive}
														title={i18n.m.admin.policies.archive}
														onclick={() => onArchive(policy)}
													>
														<Archive class="size-4" />
													</button>
												{/if}
												<a
													href={`/privacy?v=${encodeURIComponent(policy.id)}&r=${policy.revision}`}
													target="_blank"
													rel="noopener"
													class="btn-icon btn-icon-sm hover:bg-surface-100-900 inline-flex"
													aria-label={i18n.m.admin.policies.preview}
													title={i18n.m.admin.policies.preview}
												>
													<Eye class="size-4" />
												</a>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				</div>
			{/if}
		{/each}

		{#if policies.length === 0}
			<p class="opacity-70 text-sm">{i18n.m.admin.policies.empty}</p>
		{/if}
	{/if}
</section>

<Modal
	open={editing !== null && editing.kind !== 'history'}
	title={editing?.kind === 'edit' ? i18n.m.admin.policies.edit : i18n.m.admin.policies.add}
	onClose={close}
>
	<form class="space-y-4" onsubmit={onSave}>
		{#if editing?.kind === 'create'}
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.policies.kindLabel}</span>
				<select required class="select" bind:value={policyKind}>
					<option value="privacy">{i18n.m.policies.kindLabel.privacy}</option>
				</select>
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.policies.slugLabel}</span>
				<input
					type="text"
					required
					bind:value={slug}
					class="input font-mono"
					placeholder="2026-08-01"
				/>
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.policies.effectiveFromLabel}</span>
				<input type="datetime-local" required class="input" bind:value={effectiveFrom} />
			</label>
		{:else if editing?.kind === 'edit'}
			<div class="text-xs opacity-70">
				{i18n.m.admin.policies.editingMeta(editing.policy.slug, editing.policy.revision)}
			</div>
			{#if editing.policy.status === 'published'}
				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.admin.policies.noteLabel}</span>
					<input
						type="text"
						bind:value={note}
						class="input"
						placeholder={i18n.m.admin.policies.notePlaceholder}
					/>
					<span class="text-xs opacity-70">{i18n.m.admin.policies.noteHelp}</span>
				</label>
			{/if}
		{/if}

		<div class="border-b border-surface-200-800 flex gap-2 text-sm">
			<button
				type="button"
				onclick={() => (activeTab = 'sv')}
				class="px-3 py-1 -mb-px border-b-2 {activeTab === 'sv'
					? 'border-primary-500 font-semibold'
					: 'border-transparent opacity-70 hover:opacity-100'}"
			>
				{i18n.m.admin.policies.tabSv}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'en')}
				class="px-3 py-1 -mb-px border-b-2 {activeTab === 'en'
					? 'border-primary-500 font-semibold'
					: 'border-transparent opacity-70 hover:opacity-100'}"
			>
				{i18n.m.admin.policies.tabEn}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'preview')}
				class="px-3 py-1 -mb-px border-b-2 {activeTab === 'preview'
					? 'border-primary-500 font-semibold'
					: 'border-transparent opacity-70 hover:opacity-100'}"
			>
				{i18n.m.admin.policies.tabPreview}
			</button>
		</div>

		{#if activeTab === 'sv'}
			<textarea
				required
				bind:value={bodySv}
				class="textarea font-mono text-sm min-h-[24rem]"
				placeholder={i18n.m.admin.policies.bodySvPlaceholder}
			></textarea>
		{:else if activeTab === 'en'}
			<textarea
				required
				bind:value={bodyEn}
				class="textarea font-mono text-sm min-h-[24rem]"
				placeholder={i18n.m.admin.policies.bodyEnPlaceholder}
			></textarea>
		{:else}
			<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-4 min-h-[24rem]">
				<div class="prose dark:prose-invert max-w-none text-sm">
					{@html previewHtml}
				</div>
			</div>
		{/if}

		{#if formError}
			<p class="text-sm text-error-600 dark:text-error-300">{formError}</p>
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
	open={editing?.kind === 'history'}
	title={editing?.kind === 'history'
		? i18n.m.admin.policies.historyHeading(editing.policy.slug)
		: ''}
	onClose={close}
>
	{#if editing?.kind === 'history'}
		<div class="space-y-3 max-h-[60vh] overflow-y-auto">
			{#each editing.revisions as rev (rev.revision)}
				<article class="border border-surface-200-800 rounded-lg p-3 space-y-1 text-sm">
					<header class="flex items-center justify-between">
						<div class="font-mono text-xs">
							rev {rev.revision}
							{#if rev.published}
								<span class="badge preset-filled-success-500 ml-2">{i18n.m.admin.policies.revLive}</span>
							{/if}
						</div>
						<div class="opacity-70 text-xs">{fmt(rev.editedAt)}</div>
					</header>
					{#if rev.note}
						<p class="opacity-80 italic">"{rev.note}"</p>
					{/if}
					<a
						href={`/privacy?v=${encodeURIComponent(rev.policyId)}&r=${rev.revision}`}
						target="_blank"
						rel="noopener"
						class="text-primary-700 dark:text-primary-300 underline text-xs"
					>
						{i18n.m.admin.policies.revOpen}
					</a>
				</article>
			{/each}
			{#if editing.revisions.length === 0}
				<p class="opacity-70">{i18n.m.admin.policies.revEmpty}</p>
			{/if}
		</div>
	{/if}
</Modal>
