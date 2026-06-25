<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import CheckCircle2 from '@lucide/svelte/icons/check-circle-2';
	import AlertCircle from '@lucide/svelte/icons/alert-circle';
	import { dsrRestore } from '$lib/api/dsr';
	import { i18n } from '$lib/i18n/state.svelte';

	type Phase = 'verifying' | 'success' | 'error';

	let phase: Phase = $state('verifying');
	let errorMsg = $state<string | null>(null);

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		if (!token) {
			phase = 'error';
			errorMsg = i18n.m.myData.restoreMissingToken;
			return;
		}
		try {
			await dsrRestore(token);
			phase = 'success';
		} catch (err) {
			phase = 'error';
			errorMsg = err instanceof Error ? err.message : String(err);
		}
	});

	async function continueToDashboard() {
		await goto(resolve('/my-data'));
	}
</script>

<section class="mx-auto max-w-xl py-12 px-4">
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-4">
		{#if phase === 'verifying'}
			<p class="opacity-70 text-sm">{i18n.m.myData.restoreVerifying}</p>
		{:else if phase === 'success'}
			<div class="flex items-start gap-3">
				<CheckCircle2 class="size-6 text-success-600 dark:text-success-300 shrink-0 mt-0.5" />
				<div>
					<h1 class="text-xl font-semibold" style="font-family: var(--heading-font-family);">
						{i18n.m.myData.restoreSuccessHeading}
					</h1>
					<p class="opacity-80 mt-1 text-sm">{i18n.m.myData.restoreSuccessBody}</p>
				</div>
			</div>
			<button type="button" class="btn preset-filled-primary-500" onclick={continueToDashboard}>
				{i18n.m.myData.restoreContinue}
			</button>
		{:else}
			<div class="flex items-start gap-3">
				<AlertCircle class="size-6 text-error-600 dark:text-error-300 shrink-0 mt-0.5" />
				<div>
					<h1 class="text-xl font-semibold" style="font-family: var(--heading-font-family);">
						{i18n.m.myData.restoreErrorHeading}
					</h1>
					<p class="opacity-80 mt-1 text-sm">{i18n.m.myData.restoreErrorBody}</p>
					{#if errorMsg}
						<p class="opacity-60 mt-2 text-xs font-mono break-all">{errorMsg}</p>
					{/if}
				</div>
			</div>
			<a href={resolve('/my-data')} class="btn preset-tonal-surface">
				{i18n.m.myData.restoreBack}
			</a>
		{/if}
	</div>
</section>
