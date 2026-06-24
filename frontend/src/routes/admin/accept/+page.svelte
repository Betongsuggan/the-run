<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import MailCheck from '@lucide/svelte/icons/mail-check';
	import { i18n } from '$lib/i18n/state.svelte';
	import { acceptInvite } from '$lib/admin/api';

	let password = $state('');
	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);

	const token = $derived($page.url.searchParams.get('token') ?? '');

	async function onSubmit(e: SubmitEvent) {
		e.preventDefault();
		errorMsg = null;
		if (!token) {
			errorMsg = i18n.m.admin.accept.missingToken;
			return;
		}
		submitting = true;
		try {
			await acceptInvite({ token, password });
			await goto(resolve('/admin'));
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			submitting = false;
		}
	}
</script>

<div class="mx-auto max-w-md py-12">
	<div
		class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 sm:p-8 space-y-5"
	>
		<header class="space-y-1">
			<div class="flex items-center gap-2 text-primary-600 dark:text-primary-300">
				<MailCheck class="size-5" />
				<span class="text-xs uppercase tracking-wide font-semibold">{i18n.m.admin.title}</span>
			</div>
			<h1
				class="text-2xl font-semibold leading-tight tracking-tight"
				style="font-family: var(--heading-font-family);"
			>
				{i18n.m.admin.accept.heading}
			</h1>
			<p class="text-sm opacity-80">{i18n.m.admin.accept.subheading}</p>
		</header>

		{#if !token}
			<p class="text-sm text-error-600 dark:text-error-300">
				{i18n.m.admin.accept.missingToken}
			</p>
		{:else}
			<form class="space-y-4" onsubmit={onSubmit}>
				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.admin.accept.passwordLabel}</span>
					<input
						type="password"
						autocomplete="new-password"
						required
						minlength={6}
						bind:value={password}
						class="input"
					/>
					<span class="text-xs opacity-70">{i18n.m.admin.accept.passwordHint}</span>
				</label>

				{#if errorMsg}
					<p class="text-sm text-error-600 dark:text-error-300">{errorMsg}</p>
				{/if}

				<button type="submit" disabled={submitting} class="btn preset-filled-primary-500 w-full">
					{submitting ? i18n.m.admin.accept.submitting : i18n.m.admin.accept.submit}
				</button>
			</form>
		{/if}
	</div>
</div>
