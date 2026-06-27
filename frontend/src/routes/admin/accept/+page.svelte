<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { i18n } from '$lib/i18n/state.svelte';
	import {
		fetchAcceptPreview,
		submitAcceptInvitation,
		type AcceptPreview
	} from '$lib/admin/api';

	type Stage =
		| { kind: 'loading' }
		| { kind: 'not-found'; message: string }
		| { kind: 'form'; preview: AcceptPreview }
		| { kind: 'done'; email: string };

	let stage = $state<Stage>({ kind: 'loading' });
	let password = $state('');
	let submitting = $state(false);
	let formError = $state<string | null>(null);

	// The invitation token lives in the URL — it's the proof of access.
	const token = $derived($page.url.searchParams.get('token') ?? '');

	onMount(async () => {
		if (!token) {
			stage = { kind: 'not-found', message: i18n.m.admin.accept.notFound };
			return;
		}
		try {
			const preview = await fetchAcceptPreview(token);
			stage = { kind: 'form', preview };
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			stage = { kind: 'not-found', message: msg || i18n.m.admin.accept.notFound };
		}
	});

	async function onSubmit(ev: SubmitEvent) {
		ev.preventDefault();
		if (stage.kind !== 'form') return;
		submitting = true;
		formError = null;
		try {
			const res = await submitAcceptInvitation(token, password);
			stage = { kind: 'done', email: res.email };
		} catch (err) {
			formError = err instanceof Error ? err.message : String(err);
		} finally {
			submitting = false;
		}
	}
</script>

<div class="mx-auto max-w-md py-12">
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 sm:p-8 space-y-4">
		<h1
			class="text-2xl font-semibold leading-tight tracking-tight"
			style="font-family: var(--heading-font-family);"
		>
			{stage.kind === 'done'
				? i18n.m.admin.accept.afterAcceptedHeading
				: i18n.m.admin.accept.heading}
		</h1>

		{#if stage.kind === 'loading'}
			<p class="text-sm opacity-80">{i18n.m.admin.accept.loading}</p>
		{:else if stage.kind === 'not-found'}
			<p class="text-sm opacity-80">{stage.message}</p>
			<a href={resolve('/admin')} class="btn preset-filled-primary-500 inline-block">
				{i18n.m.admin.accept.backToLogin}
			</a>
		{:else if stage.kind === 'form'}
			<p class="text-sm opacity-80">
				{i18n.m.admin.accept.invitationFor(stage.preview.email)}
			</p>
			{#if stage.preview.invitedByMail}
				<p class="text-xs opacity-70">
					{i18n.m.admin.accept.invitedBy(stage.preview.invitedByMail)}
				</p>
			{/if}
			<form class="space-y-4" onsubmit={onSubmit}>
				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.admin.accept.passwordLabel}</span>
					<input
						type="password"
						autocomplete="new-password"
						required
						minlength={12}
						bind:value={password}
						class="input"
					/>
					<span class="text-xs opacity-70">{i18n.m.admin.accept.passwordHelp}</span>
				</label>

				{#if formError}
					<p class="text-sm text-error-600 dark:text-error-300">{formError}</p>
				{/if}

				<button type="submit" disabled={submitting} class="btn preset-filled-primary-500 w-full">
					{submitting ? i18n.m.admin.accept.submitting : i18n.m.admin.accept.submit}
				</button>
			</form>
		{:else if stage.kind === 'done'}
			<p class="text-sm opacity-80">{i18n.m.admin.accept.afterAccepted}</p>
			<a href={resolve('/admin')} class="btn preset-filled-primary-500 inline-block">
				{i18n.m.admin.accept.backToLogin}
			</a>
		{/if}
	</div>
</div>
