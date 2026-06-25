<script lang="ts">
	import LogIn from '@lucide/svelte/icons/log-in';
	import { i18n } from '$lib/i18n/state.svelte';
	import { login, MOCK_SEED_ADMIN } from '$lib/admin/api';

	let email = $state('');
	let password = $state('');
	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);

	async function onSubmit(e: SubmitEvent) {
		e.preventDefault();
		submitting = true;
		errorMsg = null;
		try {
			await login(email, password);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			submitting = false;
		}
	}
</script>

<div class="mx-auto max-w-md py-12">
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 sm:p-8 space-y-5">
		<header class="space-y-1">
			<div class="flex items-center gap-2 text-primary-600 dark:text-primary-300">
				<LogIn class="size-5" />
				<span class="text-xs uppercase tracking-wide font-semibold">{i18n.m.admin.title}</span>
			</div>
			<h1
				class="text-2xl font-semibold leading-tight tracking-tight"
				style="font-family: var(--heading-font-family);"
			>
				{i18n.m.admin.login.heading}
			</h1>
			<p class="text-sm opacity-80">{i18n.m.admin.login.subheading}</p>
		</header>

		<form class="space-y-4" onsubmit={onSubmit}>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.login.emailLabel}</span>
				<input type="email" autocomplete="email" required bind:value={email} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.admin.login.passwordLabel}</span>
				<input
					type="password"
					autocomplete="current-password"
					required
					bind:value={password}
					class="input"
				/>
			</label>

			{#if errorMsg}
				<p class="text-sm text-error-600 dark:text-error-300">{errorMsg}</p>
			{/if}

			<button type="submit" disabled={submitting} class="btn preset-filled-primary-500 w-full">
				{submitting ? i18n.m.admin.login.submitting : i18n.m.admin.login.submit}
			</button>
		</form>

		<p class="text-xs opacity-70 border-t border-surface-200-800 pt-4">
			{i18n.m.admin.login.mockHint(MOCK_SEED_ADMIN.email, MOCK_SEED_ADMIN.password)}
		</p>
	</div>
</div>
