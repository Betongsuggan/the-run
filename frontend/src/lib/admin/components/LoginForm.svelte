<script lang="ts">
	import LogIn from '@lucide/svelte/icons/log-in';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import { i18n } from '$lib/i18n/state.svelte';
	import { auth } from '$lib/admin/auth.svelte';

	let email = $state('');
	let password = $state('');
	let code = $state('');
	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);

	async function onPasswordSubmit(e: SubmitEvent) {
		e.preventDefault();
		submitting = true;
		errorMsg = null;
		try {
			await auth.login(email, password);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			submitting = false;
		}
	}

	async function onTotpSubmit(e: SubmitEvent) {
		e.preventDefault();
		submitting = true;
		errorMsg = null;
		try {
			await auth.verifyTotp(code.trim());
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
				{#if auth.step === 'totp_verify' || auth.step === 'totp_enroll'}
					<ShieldCheck class="size-5" />
				{:else}
					<LogIn class="size-5" />
				{/if}
				<span class="text-xs uppercase tracking-wide font-semibold">{i18n.m.admin.title}</span>
			</div>
			<h1
				class="text-2xl font-semibold leading-tight tracking-tight"
				style="font-family: var(--heading-font-family);"
			>
				{#if auth.step === 'totp_enroll'}
					{i18n.m.admin.login.enrollHeading}
				{:else if auth.step === 'totp_verify'}
					{i18n.m.admin.login.verifyHeading}
				{:else}
					{i18n.m.admin.login.heading}
				{/if}
			</h1>
			<p class="text-sm opacity-80">
				{#if auth.step === 'totp_enroll'}
					{i18n.m.admin.login.enrollSubheading}
				{:else if auth.step === 'totp_verify'}
					{i18n.m.admin.login.verifySubheading}
				{:else}
					{i18n.m.admin.login.subheading}
				{/if}
			</p>
		</header>

		{#if auth.step === 'password'}
			<form class="space-y-4" onsubmit={onPasswordSubmit}>
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
		{:else if auth.step === 'totp_enroll' || auth.step === 'totp_verify'}
			{#if auth.step === 'totp_enroll' && auth.enroll}
				<div class="space-y-3 text-sm">
					<p>{i18n.m.admin.login.enrollInstructions}</p>
					<code
						class="block break-all rounded-md bg-surface-200-800 px-3 py-2 text-xs font-mono opacity-90"
					>
						{auth.enroll.otpauthUri}
					</code>
					<p class="opacity-80">
						{i18n.m.admin.login.enrollSecretLabel}
						<code class="font-mono">{auth.enroll.secret}</code>
					</p>
				</div>
			{/if}
			<form class="space-y-4" onsubmit={onTotpSubmit}>
				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.admin.login.codeLabel}</span>
					<input
						type="text"
						inputmode="numeric"
						pattern="[0-9]*"
						autocomplete="one-time-code"
						required
						maxlength="8"
						bind:value={code}
						class="input"
					/>
				</label>

				{#if errorMsg}
					<p class="text-sm text-error-600 dark:text-error-300">{errorMsg}</p>
				{/if}

				<button type="submit" disabled={submitting} class="btn preset-filled-primary-500 w-full">
					{submitting ? i18n.m.admin.login.submitting : i18n.m.admin.login.codeSubmit}
				</button>
			</form>
		{/if}
	</div>
</div>
