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
	import Trash2 from '@lucide/svelte/icons/trash-2';
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
		dsrEraseRunner,
		dsrEraseAccount,
		dsrCancelDeletion,
		type DSRMe,
		type DSRRunner,
		type DSRAuditRow
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

	// Erase modal state. `eraseTarget` is either {kind: 'runner', runner}
	// or {kind: 'account'}. Null means the modal is closed.
	type EraseTarget = { kind: 'runner'; runner: DSRRunner } | { kind: 'account' };
	let eraseTarget: EraseTarget | null = $state(null);
	let eraseConfirm = $state('');
	let eraseError = $state<string | null>(null);
	let erasedAccount = $state(false); // post-account-deletion success card

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

	// ── Cancel pending deletion ─────────────────────────────────────────
	// Counterpart to the dedicated /my-data/restore route, but for users
	// who logged in normally and want to back out of a prior delete from
	// inside the dashboard.

	async function onCancelDeletion() {
		if (busy) return;
		busy = true;
		try {
			me = await dsrCancelDeletion();
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	// formatRetentionDate handles the YYYY-MM-DD strings the backend
	// emits for the retention block (date-only, not full ISO timestamps).
	function formatRetentionDate(d: string): string {
		const dt = new Date(d + 'T00:00:00Z');
		if (Number.isNaN(dt.getTime())) return d;
		return dt.toLocaleDateString(i18n.locale === 'sv' ? 'sv-SE' : 'en-GB', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}

	function formatDeletionDate(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return iso;
		return d.toLocaleDateString(i18n.locale === 'sv' ? 'sv-SE' : 'en-GB', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}

	// ── Erase ───────────────────────────────────────────────────────────
	// Typed-confirmation modal. User must type "RADERA" (sv) / "DELETE"
	// (en) — we accept either so locale switches mid-flow don't break it.

	const ERASE_CONFIRM_TOKENS = ['RADERA', 'DELETE'];

	function openEraseRunner(r: DSRRunner) {
		eraseTarget = { kind: 'runner', runner: r };
		eraseConfirm = '';
		eraseError = null;
	}
	function openEraseAccount() {
		eraseTarget = { kind: 'account' };
		eraseConfirm = '';
		eraseError = null;
	}
	function closeErase() {
		eraseTarget = null;
	}
	async function submitErase(e: SubmitEvent) {
		e.preventDefault();
		if (!eraseTarget || busy) return;
		if (!ERASE_CONFIRM_TOKENS.includes(eraseConfirm.trim().toUpperCase())) {
			eraseError = i18n.m.myData.eraseConfirmMismatch;
			return;
		}
		busy = true;
		eraseError = null;
		try {
			if (eraseTarget.kind === 'runner') {
				me = await dsrEraseRunner(eraseTarget.runner.id);
				eraseTarget = null;
			} else {
				await dsrEraseAccount();
				// Server cleared our session cookie. Drop local state and
				// switch to a dedicated success card — they shouldn't see
				// the dashboard anymore.
				eraseTarget = null;
				erasedAccount = true;
				me = null;
				email = '';
			}
		} catch (err) {
			eraseError = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	// ── Activity log helpers ────────────────────────────────────────────
	// activityLabel maps the machine-readable action codes from the audit
	// log to localized human-readable labels. Unknown actions fall back to
	// the row's summary (or the raw code if both are missing).

	function activityLabel(row: DSRAuditRow): string {
		const a = i18n.m.myData.activityActions as Record<string, string>;
		if (row.action in a) return a[row.action];
		return row.summary ?? row.action;
	}

	function formatActor(actor: string): string {
		if (actor === 'user') return i18n.m.myData.activityActorUser;
		if (actor === 'system') return i18n.m.myData.activityActorSystem;
		if (actor.startsWith('admin')) return i18n.m.myData.activityActorAdmin;
		return actor;
	}

	function formatActivityTimestamp(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return iso;
		// Locale-aware short date + time. SvelteKit's i18n.locale matches
		// the page locale so the date format follows.
		return d.toLocaleString(i18n.locale === 'sv' ? 'sv-SE' : 'en-GB', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
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
		<!-- Pending-deletion banner. Surfaces at the very top so the user
		     can't miss that their data is queued for erasure. Cancel is a
		     single click — no confirmation modal — since the destructive
		     action was already confirmed when they originally deleted. -->
		{#if me.account.deletionPendingUntil}
			<div
				class="card border border-error-300 dark:border-error-900/60 bg-error-50/40 dark:bg-error-950/30 p-4 sm:p-5 flex items-start gap-3"
			>
				<AlertCircle class="size-5 text-error-600 dark:text-error-300 shrink-0 mt-0.5" />
				<div class="min-w-0 flex-1 space-y-2">
					<p class="font-semibold text-error-700 dark:text-error-300">
						{i18n.m.myData.pendingDeletionHeading(
							formatDeletionDate(me.account.deletionPendingUntil)
						)}
					</p>
					<p class="text-sm opacity-80">{i18n.m.myData.pendingDeletionBody}</p>
					<button
						type="button"
						class="btn preset-filled-primary-500"
						onclick={onCancelDeletion}
						disabled={busy}
					>
						{i18n.m.myData.pendingDeletionCancel}
					</button>
				</div>
			</div>
		{/if}

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
					{#if me.account.marketingConsent?.policyVersion}
						{@const consent = me.account.marketingConsent}
						<div class="text-xs opacity-60 pl-6 pt-0.5">
							{#if consent.policyId && consent.policyRevision}
								<a
									href={`/privacy?v=${encodeURIComponent(consent.policyId)}&r=${consent.policyRevision}`}
									class="underline hover:text-primary-700 dark:hover:text-primary-300"
								>
									{i18n.m.myData.consentStamp(
										consent.at.slice(0, 10),
										consent.policyVersion,
										consent.policyRevision
									)}
								</a>
							{:else}
								{i18n.m.myData.consentStampLegacy(
									consent.at.slice(0, 10),
									consent.policyVersion
								)}
							{/if}
						</div>
					{/if}
				</dd>
			</dl>
			{#if me.account.inactivityDeletionAt && !me.account.deletionPendingUntil}
				<!-- Inactivity countdown — surfaced only when the account isn't
				     already in a pending-deletion state (the banner above
				     covers that). Italic + muted so it sits as soft
				     informational footer rather than alarming the user. -->
				<p class="text-xs italic opacity-70 pt-3 mt-3 border-t border-surface-200-800">
					{i18n.m.myData.inactivityNote(
						formatRetentionDate(me.account.inactivityDeletionAt.slice(0, 10))
					)}
				</p>
			{/if}
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
									<button
										type="button"
										class="btn-icon btn-icon-sm opacity-70 hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
										aria-label={i18n.m.myData.eraseRunner}
										title={i18n.m.myData.eraseRunner}
										onclick={() => openEraseRunner(r)}
									>
										<Trash2 class="size-3.5" />
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
							{#if r.publicResultsConsent?.policyVersion}
								{@const consent = r.publicResultsConsent}
								<div class="text-xs opacity-60 pl-6">
									{#if consent.policyId && consent.policyRevision}
										<a
											href={`/privacy?v=${encodeURIComponent(consent.policyId)}&r=${consent.policyRevision}`}
											class="underline hover:text-primary-700 dark:hover:text-primary-300"
										>
											{i18n.m.myData.consentStamp(
												consent.at.slice(0, 10),
												consent.policyVersion,
												consent.policyRevision
											)}
										</a>
									{:else}
										{i18n.m.myData.consentStampLegacy(
											consent.at.slice(0, 10),
											consent.policyVersion
										)}
									{/if}
								</div>
							{/if}
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
							<!-- Retention status. Differentiated wording per case:
							     consenting runners are told their history is kept;
							     non-consenting ones see the exact date PII goes away.
							     New runners (no finished races) see a soft "no timer
							     yet" so they know the policy exists without alarm. -->
							<div class="text-xs opacity-70 pt-0.5 border-t border-surface-200-800 mt-2 pt-2">
								{#if r.retention.policy === 'kept-indefinitely'}
									{i18n.m.myData.retentionIndefinite}
								{:else if r.retention.policy === 'anonymizes-at' && r.retention.anonymizesAt}
									{i18n.m.myData.retentionAnonymizesAt(
										formatRetentionDate(r.retention.anonymizesAt)
									)}
								{:else}
									{i18n.m.myData.retentionNoTimer}
								{/if}
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<!-- Your activity — chronological audit log of what we've done with
		     your data. We render it whenever there's either (a) at least
		     one audit row OR (b) a pending deletion to surface as an
		     upcoming event. That second case covers accounts deleted
		     before the audit recorder shipped, plus the general "this is
		     the next thing scheduled to happen to your data" framing. -->
		{#if (me.recentAudit && me.recentAudit.length > 0) || me.account.deletionPendingUntil}
			<section
				class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-3"
			>
				<h2 class="text-lg font-semibold">{i18n.m.myData.activityHeading}</h2>
				<p class="text-xs opacity-70">{i18n.m.myData.activityBody}</p>
				<ol class="space-y-2 text-sm">
					{#if me.account.deletionPendingUntil}
						<!-- Upcoming event — visually distinct (error tone +
						     dashed border) so it's clear this hasn't happened
						     yet but is on the schedule. -->
						<li
							class="flex items-start gap-3 border-l-2 border-dashed border-error-400 dark:border-error-500 pl-3 py-0.5"
						>
							<time class="font-mono text-xs opacity-60 shrink-0 w-32 pt-0.5">
								{formatActivityTimestamp(me.account.deletionPendingUntil)}
							</time>
							<div class="min-w-0 flex-1">
								<div class="text-error-700 dark:text-error-300">
									{i18n.m.myData.activityUpcomingDeletion}
								</div>
								<div class="text-[10px] opacity-50 uppercase tracking-wide mt-0.5">
									{i18n.m.myData.activityUpcoming}
								</div>
							</div>
						</li>
					{/if}
					{#if me.recentAudit && me.recentAudit.length > 0}
						{#each me.recentAudit as row, idx (row.at + '-' + idx)}
							<li class="flex items-start gap-3 border-l-2 border-surface-300-700 pl-3 py-0.5">
								<time class="font-mono text-xs opacity-60 shrink-0 w-32 pt-0.5">
									{formatActivityTimestamp(row.at)}
								</time>
								<div class="min-w-0 flex-1">
									<div>{activityLabel(row)}</div>
									{#if row.summary && row.summary !== activityLabel(row)}
										<div class="text-xs opacity-70 mt-0.5">{row.summary}</div>
									{/if}
									<div class="text-[10px] opacity-50 uppercase tracking-wide mt-0.5">
										{formatActor(row.actor)}
									</div>
								</div>
							</li>
						{/each}
					{/if}
				</ol>
			</section>
		{/if}

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

		<!-- Destructive zone — bordered in error tones so it's visually
		     distinct from the normal-action cards above. -->
		<section
			class="card border border-error-300 dark:border-error-900/60 bg-error-50/40 dark:bg-error-950/30 p-6 space-y-3"
		>
			<h2 class="text-lg font-semibold text-error-700 dark:text-error-300">
				{i18n.m.myData.deleteAccountHeading}
			</h2>
			<p class="text-sm opacity-80">{i18n.m.myData.deleteAccountBody}</p>
			<button
				type="button"
				class="btn preset-filled-error-500 inline-flex items-center gap-2"
				onclick={openEraseAccount}
			>
				<Trash2 class="size-4" />
				{i18n.m.myData.deleteAccountSubmit}
			</button>
		</section>
	{:else if erasedAccount}
		<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-3">
			<div class="flex items-start gap-3">
				<CheckCircle2 class="size-6 text-success-600 dark:text-success-300 shrink-0 mt-0.5" />
				<div>
					<h2 class="text-lg font-semibold">{i18n.m.myData.deleteAccountDoneHeading}</h2>
					<p class="opacity-80 mt-1 text-sm">{i18n.m.myData.deleteAccountDoneBody}</p>
				</div>
			</div>
		</div>
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

<!-- Erase modal — used by both the per-runner Trash icon and the
     destructive-zone "Delete account" button. The form's typed-confirmation
     blocks accidental fires; the backend's 30-day grace + restore email
     blocks regret. -->
<Modal
	open={eraseTarget !== null}
	title={eraseTarget?.kind === 'runner'
		? i18n.m.myData.eraseRunnerHeading
		: i18n.m.myData.eraseAccountHeading}
	onClose={closeErase}
>
	{#if eraseTarget}
		<form class="space-y-4" onsubmit={submitErase}>
			<p class="text-sm">
				{#if eraseTarget.kind === 'runner'}
					{i18n.m.myData.eraseRunnerBody(eraseTarget.runner.name)}
				{:else}
					{i18n.m.myData.eraseAccountBody}
				{/if}
			</p>
			<ul class="text-xs opacity-80 list-disc list-inside space-y-1">
				<li>{i18n.m.myData.eraseBulletGrace}</li>
				<li>{i18n.m.myData.eraseBulletRestoreLink}</li>
				<li>{i18n.m.myData.eraseBulletHistory}</li>
			</ul>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{i18n.m.myData.eraseConfirmLabel}</span>
				<input
					type="text"
					required
					bind:value={eraseConfirm}
					class="input font-mono"
					autocomplete="off"
					placeholder="RADERA / DELETE"
				/>
			</label>
			{#if eraseError}
				<p class="text-sm text-error-600 dark:text-error-300">{eraseError}</p>
			{/if}
			<div class="flex items-center justify-end gap-2 pt-2">
				<button type="button" class="btn preset-tonal-surface" onclick={closeErase} disabled={busy}>
					{i18n.m.myData.cancel}
				</button>
				<button type="submit" class="btn preset-filled-error-500" disabled={busy}>
					{i18n.m.myData.eraseSubmit}
				</button>
			</div>
		</form>
	{/if}
</Modal>
