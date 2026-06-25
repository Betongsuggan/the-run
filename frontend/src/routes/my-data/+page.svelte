<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import UserCog from '@lucide/svelte/icons/user-cog';
	import Mail from '@lucide/svelte/icons/mail';
	import Download from '@lucide/svelte/icons/download';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Lock from '@lucide/svelte/icons/lock';
	import CheckCircle2 from '@lucide/svelte/icons/check-circle-2';
	import AlertCircle from '@lucide/svelte/icons/alert-circle';
	import {
		dsrRequestLink,
		dsrVerify,
		dsrMe,
		dsrLogout,
		dsrExportDownload,
		dsrPatchConsents,
		dsrPatchRunner,
		dsrPatchLocale,
		dsrRequestEmailChange,
		type DSRMe,
		type DSRRunner
	} from '$lib/api/dsr';
	import { i18n } from '$lib/i18n/state.svelte';
	import { formatDate } from '$lib/format';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';

	type Phase = 'request' | 'request-sent' | 'verifying' | 'verify-error' | 'dashboard';

	let phase: Phase = $state('request');
	let email = $state('');
	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);
	let me = $state<DSRMe | null>(null);
	let exporting = $state(false);
	let busy = $state(false);

	// Change-email modal state
	let changeEmailOpen = $state(false);
	let newEmail = $state('');
	let changeEmailSent = $state(false);
	let changeEmailError = $state<string | null>(null);

	// Edit-runner modal state
	let editingRunner: DSRRunner | null = $state(null);
	let editName = $state('');
	let editGender = $state<'M' | 'F' | 'X'>('M');
	let editDob = $state('');
	let editRunnerError = $state<string | null>(null);

	onMount(async () => {
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

	// ── Consent + locale toggles ────────────────────────────────────────
	// All run optimistically against the server. We swap in the server's
	// freshly-stamped policyVersion + timestamp on the response so the UI
	// matches the persisted state exactly.

	async function toggleMarketing(next: boolean) {
		if (busy) return;
		busy = true;
		try {
			me = await dsrPatchConsents({ marketing: next });
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	async function togglePublicResults(runnerId: string, next: boolean) {
		if (busy) return;
		busy = true;
		try {
			me = await dsrPatchConsents({ perRunner: { [runnerId]: { publicResults: next } } });
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	async function setLocale(locale: 'sv' | 'en') {
		if (busy) return;
		busy = true;
		try {
			me = await dsrPatchLocale(locale);
			// Mirror locally so the page chrome matches without a reload.
			i18n.set(locale);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	// ── Email change ────────────────────────────────────────────────────

	function openChangeEmail() {
		newEmail = '';
		changeEmailSent = false;
		changeEmailError = null;
		changeEmailOpen = true;
	}
	function closeChangeEmail() {
		changeEmailOpen = false;
	}
	async function submitChangeEmail(e: SubmitEvent) {
		e.preventDefault();
		if (busy) return;
		busy = true;
		changeEmailError = null;
		try {
			await dsrRequestEmailChange(newEmail.trim());
			changeEmailSent = true;
		} catch (err) {
			changeEmailError = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	// ── Edit runner ─────────────────────────────────────────────────────

	function runnerHasFinishedResult(runnerId: string): boolean {
		if (!me) return false;
		return me.registrations.some((r) => r.runnerId === runnerId && r.status === 'finished');
	}

	function openEditRunner(r: DSRRunner) {
		editingRunner = r;
		editName = r.name;
		editGender = (r.gender as 'M' | 'F' | 'X') ?? 'M';
		editDob = r.birthDate;
		editRunnerError = null;
	}
	function closeEditRunner() {
		editingRunner = null;
	}
	async function submitEditRunner(e: SubmitEvent) {
		e.preventDefault();
		if (!editingRunner || busy) return;
		busy = true;
		editRunnerError = null;
		const body: { name?: string; gender?: 'M' | 'F' | 'X'; dateOfBirth?: string } = {};
		if (editName.trim() !== editingRunner.name) body.name = editName.trim();
		if (editGender !== editingRunner.gender) body.gender = editGender;
		if (editDob !== editingRunner.birthDate) body.dateOfBirth = editDob;
		try {
			me = await dsrPatchRunner(editingRunner.id, body);
			editingRunner = null;
		} catch (err) {
			editRunnerError = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
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
			<dl class="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-3 text-sm items-center">
				<dt class="opacity-70">{i18n.m.myData.youEmail}</dt>
				<dd class="flex items-center gap-2">
					<span class="font-medium">{me.account.email}</span>
					<button
						type="button"
						class="btn-icon btn-icon-sm opacity-70 hover:opacity-100"
						aria-label={i18n.m.myData.changeEmail}
						title={i18n.m.myData.changeEmail}
						onclick={openChangeEmail}
					>
						<Pencil class="size-3.5" />
					</button>
				</dd>

				<dt class="opacity-70">{i18n.m.myData.youCreatedAt}</dt>
				<dd>{formatDate(me.account.createdAt.slice(0, 10))}</dd>

				<dt class="opacity-70">{i18n.m.myData.youLocale}</dt>
				<dd class="flex items-center gap-2">
					<button
						type="button"
						class="btn btn-sm {me.account.locale === 'sv'
							? 'preset-filled-primary-500'
							: 'preset-tonal-surface'}"
						onclick={() => setLocale('sv')}
						disabled={busy}
					>
						Svenska
					</button>
					<button
						type="button"
						class="btn btn-sm {me.account.locale === 'en'
							? 'preset-filled-primary-500'
							: 'preset-tonal-surface'}"
						onclick={() => setLocale('en')}
						disabled={busy}
					>
						English
					</button>
				</dd>

				<dt class="opacity-70">{i18n.m.myData.youMarketingConsent}</dt>
				<dd>
					<label class="inline-flex items-center gap-2">
						<input
							type="checkbox"
							class="checkbox"
							checked={me.account.marketingConsent?.granted ?? false}
							disabled={busy}
							onchange={(e) => toggleMarketing(e.currentTarget.checked)}
						/>
						<span class="text-xs opacity-70">
							{me.account.marketingConsent?.granted
								? i18n.m.myData.consentGranted
								: i18n.m.myData.consentNot}
						</span>
					</label>
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
						{@const runnerAge = ageFromBirthDate(r.birthDate)}
						{@const isUnder13 = runnerAge !== null && runnerAge < 13}
						<li class="rounded-md border border-surface-200-800 p-4 space-y-2 bg-surface-50-950">
							<div class="flex items-baseline justify-between gap-2 flex-wrap">
								<div class="font-medium">{r.name}</div>
								<div class="flex items-center gap-2">
									<div class="text-xs opacity-70">
										{r.gender}
										{#if ageFromBirthDate(r.birthDate) !== null}
											· {ageFromBirthDate(r.birthDate)} {i18n.m.myData.runnersYears}
										{/if}
									</div>
									<button
										type="button"
										class="btn-icon btn-icon-sm opacity-70 hover:opacity-100"
										aria-label={i18n.m.myData.editRunner}
										title={i18n.m.myData.editRunner}
										onclick={() => openEditRunner(r)}
									>
										<Pencil class="size-3.5" />
									</button>
								</div>
							</div>
							<label class="flex items-center gap-2 text-xs opacity-90 pt-1">
								<input
									type="checkbox"
									class="checkbox"
									checked={r.publicResultsConsent?.granted ?? false}
									disabled={busy}
									onchange={(e) => togglePublicResults(r.id, e.currentTarget.checked)}
								/>
								<span>{i18n.m.myData.runnersPublicResults}</span>
							</label>
							{#if isUnder13}
								<p class="text-xs italic opacity-60 pl-6">
									{i18n.m.myData.runnersUnder13Note(r.name)}
								</p>
							{/if}
							{#if me.registrations.some((reg) => reg.runnerId === r.id)}
								<div class="text-xs opacity-70 pt-0.5">
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

		{#if errorMsg}
			<p class="text-sm text-error-600 dark:text-error-300">{errorMsg}</p>
		{/if}

		<p class="text-xs opacity-60 text-center">{i18n.m.myData.moreSoon}</p>
	{/if}
</section>

<!-- Change-email modal -->
<Modal open={changeEmailOpen} title={i18n.m.myData.changeEmailHeading} onClose={closeChangeEmail}>
	{#if changeEmailSent}
		<div class="space-y-4">
			<div class="flex items-start gap-3">
				<CheckCircle2 class="size-6 text-success-600 dark:text-success-300 shrink-0 mt-0.5" />
				<p class="text-sm opacity-90">{i18n.m.myData.changeEmailSent}</p>
			</div>
			<div class="flex justify-end">
				<button type="button" class="btn preset-tonal-surface" onclick={closeChangeEmail}>
					{i18n.m.myData.close}
				</button>
			</div>
		</div>
	{:else}
		<form class="space-y-4" onsubmit={submitChangeEmail}>
			<p class="text-sm opacity-80">{i18n.m.myData.changeEmailBody}</p>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.myData.changeEmailLabel}</span>
				<input type="email" required bind:value={newEmail} class="input" autocomplete="email" />
			</label>
			{#if changeEmailError}
				<p class="text-sm text-error-600 dark:text-error-300">{changeEmailError}</p>
			{/if}
			<div class="flex items-center justify-end gap-2 pt-2">
				<button
					type="button"
					class="btn preset-tonal-surface"
					onclick={closeChangeEmail}
					disabled={busy}
				>
					{i18n.m.myData.cancel}
				</button>
				<button type="submit" class="btn preset-filled-primary-500" disabled={busy}>
					{i18n.m.myData.changeEmailSubmit}
				</button>
			</div>
		</form>
	{/if}
</Modal>

<!-- Edit-runner modal -->
<Modal
	open={editingRunner !== null}
	title={i18n.m.myData.editRunnerHeading}
	onClose={closeEditRunner}
>
	{#if editingRunner}
		{@const locked = runnerHasFinishedResult(editingRunner.id)}
		<form class="space-y-4" onsubmit={submitEditRunner}>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.myData.editRunnerName}</span>
				<input type="text" required bind:value={editName} class="input" />
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.myData.editRunnerGender}</span>
				<select bind:value={editGender} class="select">
					<option value="M">M</option>
					<option value="F">F</option>
					<option value="X">X</option>
				</select>
			</label>
			<label class="block space-y-1">
				<span class="text-sm font-medium inline-flex items-center gap-1.5">
					{i18n.m.myData.editRunnerDob}
					{#if locked}
						<Lock class="size-3.5 opacity-60" />
					{/if}
				</span>
				<input
					type="date"
					bind:value={editDob}
					max={new Date().toISOString().slice(0, 10)}
					class="input"
					disabled={locked}
				/>
				{#if locked}
					<span class="text-xs opacity-70">{i18n.m.myData.editRunnerDobLocked}</span>
				{/if}
			</label>
			{#if editRunnerError}
				<p class="text-sm text-error-600 dark:text-error-300">{editRunnerError}</p>
			{/if}
			<div class="flex items-center justify-end gap-2 pt-2">
				<button
					type="button"
					class="btn preset-tonal-surface"
					onclick={closeEditRunner}
					disabled={busy}
				>
					{i18n.m.myData.cancel}
				</button>
				<button type="submit" class="btn preset-filled-primary-500" disabled={busy}>
					{i18n.m.myData.save}
				</button>
			</div>
		</form>
	{/if}
</Modal>
