<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import ClipboardPlus from '@lucide/svelte/icons/clipboard-plus';
	import Shield from '@lucide/svelte/icons/shield';
	import CheckCircle2 from '@lucide/svelte/icons/check-circle-2';
	import { listEvents, listRacesForEvent, registerForRace } from '$lib/api';
	import type { Gender, Race, RaceEvent } from '$lib/types';
	import { i18n } from '$lib/i18n/state.svelte';
	import { formatDate, formatRaceName } from '$lib/format';
	import Hero from '$lib/components/Hero.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';

	type UpcomingRace = { race: Race; event: RaceEvent };

	let upcoming: UpcomingRace[] = $state([]);
	let loading = $state(true);

	let name = $state('');
	let email = $state('');
	let dateOfBirth = $state('');
	let gender = $state<Gender>('M');
	let raceId = $state('');
	// Public-results consent defaults to true (opt-out); marketing defaults to
	// false (opt-in). The backend stamps the policy version + timestamp.
	let publicResults = $state(true);
	let marketing = $state(false);
	// Honeypot — kept empty by real users; bots tend to fill every field.
	// TODO: layer on a real captcha (Cloudflare Turnstile or hCaptcha) before
	// launch — see PROJECT_PLAN.md.
	let website = $state('');

	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);
	let success = $state(false);

	const today = new Date().toISOString().slice(0, 10);

	function isUpcoming(event: RaceEvent): boolean {
		return event.date >= today;
	}

	onMount(async () => {
		const events = await listEvents();
		const futureEvents = events.filter(isUpcoming);
		const lists = await Promise.all(
			futureEvents.map(async (event) => {
				const races = await listRacesForEvent(event.id);
				return races.map((race) => ({ race, event }));
			})
		);
		upcoming = lists.flat().sort((a, b) => (a.event.date < b.event.date ? -1 : 1));

		const presetEventId = $page.url.searchParams.get('eventId');
		if (presetEventId) {
			const presetMatch = upcoming.find((u) => u.event.id === presetEventId);
			if (presetMatch) raceId = presetMatch.race.id;
		}
		if (!raceId && upcoming.length > 0) {
			raceId = upcoming[0].race.id;
		}
		loading = false;
	});

	async function onSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;
		submitting = true;
		errorMsg = null;
		try {
			await registerForRace({
				name: name.trim(),
				email: email.trim(),
				dateOfBirth,
				gender,
				raceId,
				publicResults,
				marketing,
				website
			});
			success = true;
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : String(err);
		} finally {
			submitting = false;
		}
	}

	function resetForAnother() {
		success = false;
		name = '';
		email = '';
		dateOfBirth = '';
		gender = 'M';
		publicResults = true;
		marketing = false;
		errorMsg = null;
	}
</script>

<section class="space-y-8">
	<Hero>
		<div class="max-w-2xl">
			<h1
				class="text-3xl sm:text-4xl font-semibold tracking-tight leading-tight"
				style="font-family: var(--heading-font-family);"
			>
				{i18n.m.register.heading}
			</h1>
			<p class="mt-3 text-base sm:text-lg opacity-85 max-w-xl">
				{i18n.m.register.subheading}
			</p>
		</div>
	</Hero>

	<section class="mx-auto max-w-xl">
		<SectionHeader title={i18n.m.register.heading} icon={ClipboardPlus} tone="primary" />

		{#if loading}
			<p class="opacity-70">{i18n.m.event.loading}</p>
		{:else if success}
			<div
				class="card preset-filled-surface-50-950 border border-success-200 dark:border-success-900/40 p-6 space-y-4"
			>
				<div class="flex items-start gap-3">
					<CheckCircle2 class="size-6 text-success-600 dark:text-success-300 shrink-0 mt-0.5" />
					<div>
						<h2 class="text-lg font-semibold">{i18n.m.register.successHeading}</h2>
						<p class="opacity-80 mt-1">{i18n.m.register.successBody}</p>
					</div>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<button type="button" class="btn preset-tonal-surface" onclick={resetForAnother}>
						{i18n.m.register.registerAnother}
					</button>
					<a href={resolve('/')} class="btn preset-filled-primary-500">
						{i18n.m.event.backHome}
					</a>
				</div>
			</div>
		{:else if upcoming.length === 0}
			<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6">
				<p class="opacity-80">{i18n.m.register.noUpcomingRaces}</p>
			</div>
		{:else}
			<form
				class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-5"
				onsubmit={onSubmit}
				autocomplete="off"
			>
				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.register.nameLabel}</span>
					<input
						type="text"
						required
						bind:value={name}
						placeholder={i18n.m.register.namePlaceholder}
						class="input"
						autocomplete="name"
					/>
				</label>

				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.register.emailLabel}</span>
					<input
						type="email"
						required
						bind:value={email}
						placeholder={i18n.m.register.emailPlaceholder}
						class="input"
						autocomplete="email"
					/>
					<span class="text-xs opacity-60">{i18n.m.register.emailHelp}</span>
				</label>

				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.register.dobLabel}</span>
					<input
						type="date"
						required
						bind:value={dateOfBirth}
						max={today}
						class="input"
						autocomplete="bday"
					/>
				</label>

				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.register.genderLabel}</span>
					<select required bind:value={gender} class="select">
						<option value="M">{i18n.m.category.gender.M}</option>
						<option value="F">{i18n.m.category.gender.F}</option>
						<option value="X">{i18n.m.category.gender.X}</option>
					</select>
				</label>

				<label class="block space-y-1">
					<span class="text-sm font-medium">{i18n.m.register.raceLabel}</span>
					<select required bind:value={raceId} class="select">
						{#each upcoming as u (u.race.id)}
							<option value={u.race.id}>
								{u.event.name} · {u.race.name} ({formatRaceName(u.race)}) — {formatDate(
									u.event.date
								)}
							</option>
						{/each}
					</select>
				</label>

				<div class="space-y-3 border-t border-surface-200-800 pt-4">
					<label class="flex items-start gap-3">
						<input type="checkbox" class="checkbox mt-0.5" bind:checked={publicResults} />
						<span class="text-sm">
							<span class="font-medium">{i18n.m.register.consentPublicResultsLabel}</span>
							<span class="block text-xs opacity-70"
								>{i18n.m.register.consentPublicResultsHelp}</span
							>
						</span>
					</label>
					<label class="flex items-start gap-3">
						<input type="checkbox" class="checkbox mt-0.5" bind:checked={marketing} />
						<span class="text-sm">
							<span class="font-medium">{i18n.m.register.consentMarketingLabel}</span>
							<span class="block text-xs opacity-70">{i18n.m.register.consentMarketingHelp}</span>
						</span>
					</label>
				</div>

				<!-- Honeypot: hidden from sighted users + assistive tech. Bots that
				     fill every input will set this; the backend silently drops them. -->
				<div aria-hidden="true" class="absolute -left-[10000px] top-auto h-px w-px overflow-hidden">
					<label>
						Website
						<input type="text" tabindex="-1" autocomplete="off" bind:value={website} />
					</label>
				</div>

				{#if errorMsg}
					<p class="text-sm text-error-600 dark:text-error-300">
						{i18n.m.register.errorPrefix}
						{errorMsg}
					</p>
				{/if}

				<div class="flex items-center justify-between gap-3 pt-1">
					<span class="text-xs opacity-60 inline-flex items-center gap-1.5">
						<Shield class="size-3.5" />
						{i18n.m.register.botProtectionNote}
					</span>
					<button type="submit" disabled={submitting} class="btn preset-filled-primary-500">
						{submitting ? i18n.m.register.submitting : i18n.m.register.submit}
					</button>
				</div>
			</form>
		{/if}
	</section>
</section>
