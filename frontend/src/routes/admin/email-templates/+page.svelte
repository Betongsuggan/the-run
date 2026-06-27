<script lang="ts">
	import { onMount } from 'svelte';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Send from '@lucide/svelte/icons/send';
	import Archive from '@lucide/svelte/icons/archive';
	import History from '@lucide/svelte/icons/history';
	import { i18n } from '$lib/i18n/state.svelte';
	import {
		adminArchiveEmailTemplate,
		adminEditEmailTemplate,
		adminListEmailTemplateRevisions,
		adminListEmailTemplates,
		adminPublishEmailTemplate
	} from '$lib/admin/api';
	import type { EmailTemplate, EmailTemplateRevision } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';

	type EditingMode =
		| { kind: 'edit'; template: EmailTemplate }
		| { kind: 'history'; template: EmailTemplate; revisions: EmailTemplateRevision[] }
		| null;

	let templates = $state<EmailTemplate[]>([]);
	let loading = $state(true);
	let listError = $state<string | null>(null);

	let editing = $state<EditingMode>(null);

	// Form state — reused by Edit form. There is no Create flow: the slug
	// set is fixed in the backend and seeded on first boot.
	let subjectSv = $state('');
	let bodySv = $state('');
	let subjectEn = $state('');
	let bodyEn = $state('');
	let note = $state('');
	let saving = $state(false);
	let formError = $state<string | null>(null);
	let activeTab = $state<'sv' | 'en' | 'preview'>('sv');

	// Sample values used to render the preview. Keep keys aligned with the
	// AvailableVariables list per template; unknown vars render as empty
	// strings (Go text/template `missingkey=zero`).
	const sampleVars: Record<string, string> = {
		RunnerName: 'Alex Johansson',
		Link: 'https://running.rydback.net/example?token=PREVIEW',
		DeletionDate: '2026-12-31',
		InvitedByMail: 'admin@example.com'
	};

	onMount(() => {
		void reload();
	});

	async function reload() {
		loading = true;
		listError = null;
		try {
			templates = await adminListEmailTemplates();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	function openEdit(template: EmailTemplate) {
		editing = { kind: 'edit', template };
		subjectSv = template.subjectSv;
		bodySv = template.bodySv;
		subjectEn = template.subjectEn;
		bodyEn = template.bodyEn;
		note = '';
		formError = null;
		activeTab = 'sv';
	}

	async function openHistory(template: EmailTemplate) {
		try {
			const revisions = await adminListEmailTemplateRevisions(template.slug);
			editing = { kind: 'history', template, revisions };
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
			await adminEditEmailTemplate(editing.template.slug, {
				subjectSv,
				bodySv,
				subjectEn,
				bodyEn,
				note: note.trim() || undefined
			});
			close();
			await reload();
		} catch (err) {
			formError = err instanceof Error ? err.message : String(err);
		} finally {
			saving = false;
		}
	}

	async function onPublish(template: EmailTemplate) {
		if (!confirm(i18n.m.admin.emailTemplates.confirmPublish(template.slug))) return;
		try {
			await adminPublishEmailTemplate(template.slug);
			await reload();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		}
	}

	async function onArchive(template: EmailTemplate) {
		if (!confirm(i18n.m.admin.emailTemplates.confirmArchive(template.slug))) return;
		try {
			await adminArchiveEmailTemplate(template.slug);
			await reload();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		}
	}

	function statusBadgeClass(status: EmailTemplate['status']): string {
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

	// Render a template body with the sample variables substituted. Mirrors
	// Go's text/template `{{.Field}}` syntax — basic regex is enough for a
	// preview; the server is authoritative on render.
	function renderPreview(body: string): string {
		return body.replace(/{{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*}}/g, (_, name) => sampleVars[name] ?? '');
	}

	const previewSubject = $derived(
		editing?.kind === 'edit'
			? renderPreview(activeTab === 'en' ? subjectEn : subjectSv)
			: ''
	);
	const previewBody = $derived(
		editing?.kind === 'edit' ? renderPreview(activeTab === 'en' ? bodyEn : bodySv) : ''
	);

	const editingTemplate = $derived(
		editing && editing.kind !== 'history' ? editing.template : null
	);
</script>

<section class="space-y-6">
	<PageHeader
		title={i18n.m.admin.emailTemplates.heading}
		description={i18n.m.admin.emailTemplates.description}
	/>

	{#if listError}
		<p class="text-sm text-error-600 dark:text-error-300">{listError}</p>
	{/if}

	{#if loading}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.loading}</p>
	{:else if templates.length === 0}
		<p class="opacity-70 text-sm">{i18n.m.admin.emailTemplates.empty}</p>
	{:else}
		<div
			class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden"
		>
			<div class="table-wrap">
				<table class="table">
					<thead>
						<tr>
							<th>{i18n.m.admin.emailTemplates.columnName}</th>
							<th>{i18n.m.admin.emailTemplates.columnSlug}</th>
							<th>{i18n.m.admin.emailTemplates.columnStatus}</th>
							<th>{i18n.m.admin.emailTemplates.columnRevision}</th>
							<th>{i18n.m.admin.emailTemplates.columnUpdated}</th>
							<th class="w-56 text-right">{i18n.m.admin.common.actions}</th>
						</tr>
					</thead>
					<tbody>
						{#each templates as t (t.slug)}
							<tr>
								<td class="text-sm">{t.displayName || t.slug}</td>
								<td class="font-mono text-xs">{t.slug}</td>
								<td>
									<span class={statusBadgeClass(t.status)}
										>{i18n.m.admin.emailTemplates.statusLabel(t.status)}</span
									>
								</td>
								<td>{t.revision}</td>
								<td>{fmt(t.updatedAt)}</td>
								<td class="text-right space-x-1">
									<button
										type="button"
										class="btn-icon btn-icon-sm hover:bg-surface-100-900"
										aria-label={i18n.m.admin.emailTemplates.history}
										title={i18n.m.admin.emailTemplates.history}
										onclick={() => openHistory(t)}
									>
										<History class="size-4" />
									</button>
									{#if t.status !== 'archived'}
										<button
											type="button"
											class="btn-icon btn-icon-sm hover:bg-surface-100-900"
											aria-label={i18n.m.admin.common.edit}
											title={i18n.m.admin.common.edit}
											onclick={() => openEdit(t)}
										>
											<Pencil class="size-4" />
										</button>
									{/if}
									{#if t.status === 'draft'}
										<button
											type="button"
											class="btn-icon btn-icon-sm hover:bg-success-100 hover:text-success-700 dark:hover:bg-success-900/40 dark:hover:text-success-200"
											aria-label={i18n.m.admin.emailTemplates.publish}
											title={i18n.m.admin.emailTemplates.publish}
											onclick={() => onPublish(t)}
										>
											<Send class="size-4" />
										</button>
									{/if}
									{#if t.status === 'published'}
										<button
											type="button"
											class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
											aria-label={i18n.m.admin.emailTemplates.archive}
											title={i18n.m.admin.emailTemplates.archive}
											onclick={() => onArchive(t)}
										>
											<Archive class="size-4" />
										</button>
									{/if}
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
	open={editing !== null && editing.kind !== 'history'}
	title={editingTemplate
		? i18n.m.admin.emailTemplates.editHeading(editingTemplate.displayName || editingTemplate.slug)
		: ''}
	onClose={close}
>
	<form class="space-y-4" onsubmit={onSave}>
		{#if editingTemplate}
			<div class="text-xs opacity-70">
				{i18n.m.admin.emailTemplates.editingMeta(editingTemplate.slug, editingTemplate.revision)}
			</div>

			{#if editingTemplate.availableVariables.length > 0}
				<div class="card preset-tonal-surface p-3 text-xs space-y-1">
					<div class="font-semibold uppercase tracking-wide opacity-80">
						{i18n.m.admin.emailTemplates.availableVariables}
					</div>
					<ul class="font-mono">
						{#each editingTemplate.availableVariables as v (v)}
							<li>{`{{.${v}}}`}</li>
						{/each}
					</ul>
					<p class="opacity-70 not-italic">
						{i18n.m.admin.emailTemplates.availableVariablesHelp}
					</p>
				</div>
			{/if}

			{#if editingTemplate.status === 'published'}
				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.admin.emailTemplates.noteLabel}</span>
					<input
						type="text"
						bind:value={note}
						class="input"
						placeholder={i18n.m.admin.emailTemplates.notePlaceholder}
					/>
					<span class="text-xs opacity-70">{i18n.m.admin.emailTemplates.noteHelp}</span>
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
				{i18n.m.admin.emailTemplates.tabSv}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'en')}
				class="px-3 py-1 -mb-px border-b-2 {activeTab === 'en'
					? 'border-primary-500 font-semibold'
					: 'border-transparent opacity-70 hover:opacity-100'}"
			>
				{i18n.m.admin.emailTemplates.tabEn}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'preview')}
				class="px-3 py-1 -mb-px border-b-2 {activeTab === 'preview'
					? 'border-primary-500 font-semibold'
					: 'border-transparent opacity-70 hover:opacity-100'}"
			>
				{i18n.m.admin.emailTemplates.tabPreview}
			</button>
		</div>

		{#if activeTab === 'sv'}
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.emailTemplates.subjectLabel}</span>
				<input type="text" required bind:value={subjectSv} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.emailTemplates.bodyLabel}</span>
				<textarea
					required
					bind:value={bodySv}
					class="textarea font-mono text-sm min-h-[20rem]"
				></textarea>
			</label>
		{:else if activeTab === 'en'}
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.emailTemplates.subjectLabel}</span>
				<input type="text" required bind:value={subjectEn} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.emailTemplates.bodyLabel}</span>
				<textarea
					required
					bind:value={bodyEn}
					class="textarea font-mono text-sm min-h-[20rem]"
				></textarea>
			</label>
		{:else}
			<div
				class="card preset-filled-surface-50-950 border border-surface-200-800 p-4 min-h-[20rem] space-y-3"
			>
				<div class="text-xs uppercase tracking-wide opacity-60">
					{i18n.m.admin.emailTemplates.subjectLabel}
				</div>
				<div class="font-semibold">{previewSubject}</div>
				<div class="text-xs uppercase tracking-wide opacity-60">
					{i18n.m.admin.emailTemplates.bodyLabel}
				</div>
				<pre class="whitespace-pre-wrap text-sm font-sans">{previewBody}</pre>
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
		? i18n.m.admin.emailTemplates.historyHeading(editing.template.slug)
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
								<span class="badge preset-filled-success-500 ml-2"
									>{i18n.m.admin.emailTemplates.revLive}</span
								>
							{/if}
						</div>
						<div class="opacity-70 text-xs">{fmt(rev.editedAt)}</div>
					</header>
					{#if rev.note}
						<p class="opacity-80 italic">"{rev.note}"</p>
					{/if}
					<div class="text-xs opacity-70 uppercase tracking-wide">
						{i18n.m.admin.emailTemplates.subjectLabel}
					</div>
					<div class="font-semibold text-sm">{rev.subjectSv}</div>
				</article>
			{/each}
			{#if editing.revisions.length === 0}
				<p class="opacity-70">{i18n.m.admin.emailTemplates.revEmpty}</p>
			{/if}
		</div>
	{/if}
</Modal>
