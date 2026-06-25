<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import Calendar from '@lucide/svelte/icons/calendar';
	import Users from '@lucide/svelte/icons/users';
	import ClipboardPlus from '@lucide/svelte/icons/clipboard-plus';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import { getHello, listEvents, listRunners, type Hello } from '$lib/api';
	import type { RaceEvent, Runner } from '$lib/types';
	import { i18n } from '$lib/i18n/state.svelte';
	import Hero from '$lib/components/Hero.svelte';
	import EventCard from '$lib/components/EventCard.svelte';
	import RunnerCard from '$lib/components/RunnerCard.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import SunBurst from '$lib/illustrations/SunBurst.svelte';
	import ArchipelagoSilhouette from '$lib/illustrations/ArchipelagoSilhouette.svelte';
	import WaveDivider from '$lib/illustrations/WaveDivider.svelte';

	let apiState:
		| { kind: 'loading' }
		| { kind: 'ok'; data: Hello }
		| { kind: 'err'; message: string } = $state({ kind: 'loading' });

	let runners: Runner[] = $state([]);
	let events: RaceEvent[] = $state([]);

	onMount(async () => {
		runners = await listRunners();
		events = await listEvents();
		try {
			const data = await getHello();
			apiState = { kind: 'ok', data };
		} catch (e) {
			apiState = { kind: 'err', message: e instanceof Error ? e.message : String(e) };
		}
	});
</script>

<section class="space-y-10">
	<Hero>
		{#snippet background()}
			<SunBurst class="absolute -top-12 -right-12 size-72 text-tertiary-500 opacity-70" />
			<ArchipelagoSilhouette
				class="absolute inset-x-0 bottom-0 h-24 w-full text-success-700 dark:text-success-300"
			/>
		{/snippet}
		<div class="max-w-2xl">
			<h1
				class="text-4xl sm:text-5xl font-semibold tracking-tight leading-tight"
				style="font-family: var(--heading-font-family);"
			>
				{i18n.m.nav.brand}
			</h1>
			<p class="mt-4 text-base sm:text-lg opacity-85 max-w-xl">
				{i18n.m.landing.heroTagline}
			</p>
			<div class="mt-6 flex flex-wrap gap-3">
				<a href="#events" class="btn preset-filled-primary-500 inline-flex items-center gap-2">
					<Calendar class="size-4" />
					{i18n.m.landing.ctaEvents}
				</a>
				<a
					href={resolve('/register')}
					class="btn preset-filled-secondary-500 inline-flex items-center gap-2"
				>
					<ClipboardPlus class="size-4" />
					{i18n.m.landing.ctaRegister}
				</a>
				<a
					href={resolve('/runners')}
					class="btn preset-tonal-surface border border-surface-200-800 inline-flex items-center gap-2"
				>
					<Users class="size-4" />
					{i18n.m.landing.ctaRunners}
				</a>
			</div>
		</div>
	</Hero>

	<WaveDivider class="h-10 w-full text-primary-300 dark:text-primary-700 -mt-6" />

	<section id="events">
		<SectionHeader title={i18n.m.landing.eventsHeading} icon={Calendar} tone="primary" />
		<ul class="grid gap-4 sm:grid-cols-2">
			{#each events as event, i (event.id)}
				<li>
					<EventCard {event} tone={i % 2 === 0 ? 'primary' : 'secondary'} />
				</li>
			{/each}
		</ul>
	</section>

	<section>
		<SectionHeader title={i18n.m.landing.runnersHeading} icon={Users} tone="secondary">
			{#snippet action()}
				<a
					href={resolve('/runners')}
					class="inline-flex items-center gap-1 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-200 px-3 py-1 text-sm font-medium hover:bg-primary-200 dark:hover:bg-primary-900/60 transition-colors"
				>
					{i18n.m.landing.seeAll}
					<ArrowRight class="size-3.5" />
				</a>
			{/snippet}
		</SectionHeader>
		<ul class="grid gap-3 sm:grid-cols-2">
			{#each runners.slice(0, 4) as runner (runner.id)}
				<li>
					<RunnerCard {runner} />
				</li>
			{/each}
		</ul>
	</section>

	<details class="text-xs opacity-70">
		<summary class="cursor-pointer">{i18n.m.landing.apiStatus}</summary>
		<div class="mt-2">
			{#if apiState.kind === 'loading'}
				{i18n.m.landing.loading}
			{:else if apiState.kind === 'err'}
				<span class="text-error-500">{i18n.m.landing.error(apiState.message)}</span>
			{:else}
				<span>{apiState.data.message} · {apiState.data.timestamp}</span>
			{/if}
		</div>
	</details>
</section>
