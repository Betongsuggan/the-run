<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import CheckCircle2 from '@lucide/svelte/icons/check-circle-2';
	import AlertCircle from '@lucide/svelte/icons/alert-circle';
	import { verifyGuardianConsent } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';

	type Phase = 'loading' | 'success' | 'error';

	let phase: Phase = $state('loading');
	let errorMsg = $state<string | null>(null);

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		if (!token) {
			phase = 'error';
			errorMsg = i18n.m.guardianConsent.missingToken;
			return;
		}
		try {
			await verifyGuardianConsent(token);
			phase = 'success';
		} catch (err) {
			phase = 'error';
			errorMsg = err instanceof Error ? err.message : String(err);
		}
	});
</script>

<section class="mx-auto max-w-xl py-12 px-4">
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-4">
		{#if phase === 'loading'}
			<p class="opacity-70">{i18n.m.guardianConsent.verifying}</p>
		{:else if phase === 'success'}
			<div class="flex items-start gap-3">
				<CheckCircle2 class="size-6 text-success-600 dark:text-success-300 shrink-0 mt-0.5" />
				<div>
					<h1 class="text-xl font-semibold" style="font-family: var(--heading-font-family);">
						{i18n.m.guardianConsent.successHeading}
					</h1>
					<p class="opacity-80 mt-1">{i18n.m.guardianConsent.successBody}</p>
				</div>
			</div>
			<a href={resolve('/')} class="btn preset-filled-primary-500">
				{i18n.m.event.backHome}
			</a>
		{:else}
			<div class="flex items-start gap-3">
				<AlertCircle class="size-6 text-error-600 dark:text-error-300 shrink-0 mt-0.5" />
				<div>
					<h1 class="text-xl font-semibold" style="font-family: var(--heading-font-family);">
						{i18n.m.guardianConsent.errorHeading}
					</h1>
					<p class="opacity-80 mt-1">{i18n.m.guardianConsent.errorBody}</p>
					{#if errorMsg}
						<p class="opacity-60 mt-2 text-xs font-mono break-all">{errorMsg}</p>
					{/if}
				</div>
			</div>
			<a href={resolve('/')} class="btn preset-tonal-surface">
				{i18n.m.event.backHome}
			</a>
		{/if}
	</div>
</section>
