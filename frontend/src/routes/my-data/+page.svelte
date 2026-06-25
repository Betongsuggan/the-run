<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import UserCog from '@lucide/svelte/icons/user-cog';
	import Mail from '@lucide/svelte/icons/mail';
	import Download from '@lucide/svelte/icons/download';
	import LogOut from '@lucide/svelte/icons/log-out';
	import CheckCircle2 from '@lucide/svelte/icons/check-circle-2';
	import AlertCircle from '@lucide/svelte/icons/alert-circle';
	import {
		dsrRequestLink,
		dsrVerify,
		dsrMe,
		dsrLogout,
		dsrExportDownload,
		type DSRMe
	} from '$lib/api/dsr';
	import { i18n } from '$lib/i18n/state.svelte';
	import { formatDate } from '$lib/format';
	import SectionHeader from '$lib/components/SectionHeader.svelte';

	type Phase = 'request' | 'request-sent' | 'verifying' | 'verify-error' | 'dashboard';

	let phase: Phase = $state('request');
	let email = $state('');
	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);
	let me = $state<DSRMe | null>(null);
	let exporting = $state(false);

	onMount(async () => {
		// Two entry paths:
		// 1) Magic link landing: ?token=... → exchange for session, drop the
		//    token from the URL, render dashboard.
		// 2) Cookie already present (returning user, or just exchanged): fetch
		//    /dsr/me; on 401 fall back to request-link form.
		const token = $page.url.searchParams.get('token');
		if (token) {
			phase = 'verifying';
			try {
				await dsrVerify(token);
				await goto(resolve('/my-data'), { replaceState: true });
				await loadMe();
			} catch (err) {
				phase = 'verify-error';
				errorMsg = err instanceof Error ? err.message : String(err);
			}
			return;
		}
		// No token: try the cookie. Silently fall through to request-link form
		// on 401 (typical "first visit" path).
		try {
			me = await dsrMe();
			phase = 'dashboard';
		} catch {
			phase = 'request';
		}
	});

	async function loadMe(): Promise<void> {
		try {
			me = await dsrMe();
			phase = 'dashboard';
		} catch (err) {
			phase = 'verify-error';
			errorMsg = err instanceof Error ? err.message : String(err);
		}
	}

	async function onRequestSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;
		submitting = true;
		errorMsg = null;
		try {
			await dsrRequestLink(email.trim());
			phase = 'request-sent';
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			submitting = false;
		}
	}

	async function onLogout() {
		try {
			await dsrLogout();
		} finally {
			me = null;
			email = '';
			phase = 'request';
		}
	}

	async function onExport() {
		if (exporting) return;
		exporting = true;
		try {
			await dsrExportDownload();
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			exporting = false;
		}
	}

	function ageFromBirthDate(birthDate: string): number | null {
		if (!birthDate) return null;
		const dob = new Date(birthDate);
		if (Number.isNaN(dob.getTime())) return null;
		const now = new Date();
		let age = now.getFullYear() - dob.getFullYear();
		const m = now.getMonth() - dob.getMonth();
		if (m < 0 || (m === 0 && now.getDate() < dob.getDate())) age--;
		return age;
	}
</script>

<section class="mx-auto max-w-2xl py-8 px-4 space-y-6">
	<SectionHeader title={i18n.m.myData.heading} icon={UserCog} tone="primary" />
	<p class="text-sm opacity-80 -mt-2">{i18n.m.myData.intro}</p>

	{#if phase === 'request'}
		<form
			class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-4"
			onsubmit={onRequestSubmit}
		>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.myData.emailLabel}</span>
				<input
					type="email"
					required
					bind:value={email}
					placeholder={i18n.m.myData.emailPlaceholder}
					class="input"
					autocomplete="email"
				/>
				<span class="text-xs opacity-60">{i18n.m.myData.emailHelp}</span>
			</label>
			{#if errorMsg}
				<p class="text-sm text-error-600 dark:text-error-300">{errorMsg}</p>
			{/if}
			<button
				type="submit"
				disabled={submitting}
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
			>
				<Mail class="size-4" />
				{submitting ? i18n.m.myData.requestSubmitting : i18n.m.myData.requestSubmit}
			</button>
		</form>
	{:else if phase === 'request-sent'}
		<div
			class="card preset-filled-surface-50-950 border border-success-200 dark:border-success-900/40 p-6 space-y-3"
		>
			<div class="flex items-start gap-3">
				<CheckCircle2 class="size-6 text-success-600 dark:text-success-300 shrink-0 mt-0.5" />
				<div>
					<h2 class="text-lg font-semibold">{i18n.m.myData.sentHeading}</h2>
					<p class="opacity-80 mt-1 text-sm">{i18n.m.myData.sentBody}</p>
				</div>
			</div>
		</div>
	{:else if phase === 'verifying'}
		<p class="opacity-70 text-sm">{i18n.m.myData.verifying}</p>
	{:else if phase === 'verify-error'}
		<div
			class="card preset-filled-surface-50-950 border border-error-200 dark:border-error-900/40 p-6 space-y-3"
		>
			<div class="flex items-start gap-3">
				<AlertCircle class="size-6 text-error-600 dark:text-error-300 shrink-0 mt-0.5" />
				<div>
					<h2 class="text-lg font-semibold">{i18n.m.myData.errorHeading}</h2>
					<p class="opacity-80 mt-1 text-sm">{i18n.m.myData.errorBody}</p>
					{#if errorMsg}
						<p class="opacity-60 mt-2 text-xs font-mono break-all">{errorMsg}</p>
					{/if}
				</div>
			</div>
			<button type="button" class="btn preset-tonal-surface" onclick={() => (phase = 'request')}>
				{i18n.m.myData.tryAgain}
			</button>
		</div>
	{:else if phase === 'dashboard' && me}
		<!-- You -->
		<section class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-4">
			<header class="flex items-center justify-between gap-3">
				<h2 class="text-lg font-semibold">{i18n.m.myData.youHeading}</h2>
				<button
					type="button"
					class="btn-icon btn-icon-sm opacity-70 hover:opacity-100"
					aria-label={i18n.m.myData.logout}
					title={i18n.m.myData.logout}
					onclick={onLogout}
				>
					<LogOut class="size-4" />
				</button>
			</header>
			<dl class="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-2 text-sm">
				<dt class="opacity-70">{i18n.m.myData.youEmail}</dt>
				<dd class="font-medium">{me.account.email}</dd>
				<dt class="opacity-70">{i18n.m.myData.youCreatedAt}</dt>
				<dd>{formatDate(me.account.createdAt.slice(0, 10))}</dd>
				{#if me.account.locale}
					<dt class="opacity-70">{i18n.m.myData.youLocale}</dt>
					<dd>{me.account.locale}</dd>
				{/if}
				<dt class="opacity-70">{i18n.m.myData.youMarketingConsent}</dt>
				<dd>
					{#if me.account.marketingConsent?.granted}
						{i18n.m.myData.consentGranted}
					{:else}
						{i18n.m.myData.consentNot}
					{/if}
				</dd>
			</dl>
		</section>

		<!-- Runners -->
		<section class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-4">
			<h2 class="text-lg font-semibold">{i18n.m.myData.runnersHeading(me.runners.length)}</h2>
			{#if me.runners.length === 0}
				<p class="text-sm opacity-70">{i18n.m.myData.runnersEmpty}</p>
			{:else}
				<ul class="space-y-3">
					{#each me.runners as r (r.id)}
						<li class="rounded-md border border-surface-200-800 p-4 space-y-2 bg-surface-50-950">
							<div class="flex items-baseline justify-between gap-2 flex-wrap">
								<div class="font-medium">{r.name}</div>
								<div class="text-xs opacity-70">
									{r.gender}
									{#if ageFromBirthDate(r.birthDate) !== null}
										· {ageFromBirthDate(r.birthDate)} {i18n.m.myData.runnersYears}
									{/if}
								</div>
							</div>
							<div class="text-xs opacity-70">
								{i18n.m.myData.runnersPublicResults}:
								{#if r.publicResultsConsent?.granted}
									{i18n.m.myData.consentGranted}
								{:else}
									{i18n.m.myData.consentNot}
								{/if}
							</div>
							{#if me.registrations.some((reg) => reg.runnerId === r.id)}
								<div class="text-xs opacity-70 pt-1">
									{i18n.m.myData.runnersRegistrations(
										me.registrations.filter((reg) => reg.runnerId === r.id).length
									)}
								</div>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<!-- Export -->
		<section class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-3">
			<h2 class="text-lg font-semibold">{i18n.m.myData.exportHeading}</h2>
			<p class="text-sm opacity-80">{i18n.m.myData.exportBody}</p>
			<button
				type="button"
				disabled={exporting}
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
				onclick={onExport}
			>
				<Download class="size-4" />
				{exporting ? i18n.m.myData.exportSubmitting : i18n.m.myData.exportSubmit}
			</button>
		</section>

		<p class="text-xs opacity-60 text-center">{i18n.m.myData.moreSoon}</p>
	{/if}
</section>
