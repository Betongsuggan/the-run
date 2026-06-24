<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { getHello, listEvents, listRunners, type Hello } from '$lib/api';
	import type { RaceEvent, Runner } from '$lib/types';
	import { formatDate } from '$lib/format';
	import { i18n } from '$lib/i18n/state.svelte';

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

<section class="space-y-8">
	<header>
		<h1 class="h1">{i18n.m.nav.brand}</h1>
		<p class="opacity-80 mt-2">{i18n.m.landing.description}</p>
	</header>

	<div class="card preset-tonal-primary p-4">
		<div class="font-semibold mb-1">{i18n.m.landing.mockTitle}</div>
		<p class="text-sm opacity-90">{i18n.m.landing.mockDescription}</p>
	</div>

	<section>
		<h2 class="h3 mb-3">{i18n.m.landing.eventsHeading}</h2>
		<ul class="grid gap-3 sm:grid-cols-2">
			{#each events as event (event.id)}
				<li>
					<a
						href={resolve('/events/[id]', { id: event.id })}
						class="card preset-filled-surface-100-900 border border-surface-200-800 p-4 block hover:preset-tonal-primary"
					>
						<div class="font-semibold">{event.name}</div>
						<div class="text-xs opacity-70 mt-1">
							{formatDate(event.date)}{#if event.location}
								· {event.location}
							{/if}
						</div>
					</a>
				</li>
			{/each}
		</ul>
	</section>

	<section>
		<div class="flex items-baseline justify-between mb-3">
			<h2 class="h3">{i18n.m.landing.runnersHeading}</h2>
			<a href={resolve('/runners')} class="text-sm text-primary-500 hover:underline"
				>{i18n.m.landing.seeAll}</a
			>
		</div>
		<ul class="grid gap-3 sm:grid-cols-2">
			{#each runners.slice(0, 4) as runner (runner.id)}
				<li>
					<a
						href={resolve('/runners/[id]', { id: runner.id })}
						class="card preset-filled-surface-100-900 border border-surface-200-800 p-4 block hover:preset-tonal-primary"
					>
						<div class="font-semibold">{runner.name}</div>
						<div class="text-xs opacity-70 mt-1">{runner.gender}</div>
					</a>
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
