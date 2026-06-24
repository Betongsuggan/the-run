<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import Route from '@lucide/svelte/icons/route';
	import Clock from '@lucide/svelte/icons/clock';
	import Activity from '@lucide/svelte/icons/activity';
	import Medal from '@lucide/svelte/icons/medal';
	import Sun from '@lucide/svelte/icons/sun';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import { getResult } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded } from '$lib/types';
	import {
		formatCategory,
		formatDate,
		formatDistance,
		formatPace,
		formatRaceName,
		formatTime
	} from '$lib/format';
	import SplitsTable from '$lib/components/SplitsTable.svelte';
	import StatCard from '$lib/components/StatCard.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Hero from '$lib/components/Hero.svelte';
	import RouteLine from '$lib/illustrations/RouteLine.svelte';
	import PineSprig from '$lib/illustrations/PineSprig.svelte';

	let result: ResultExpanded | undefined = $state(undefined);
	let loading = $state(true);

	$effect(() => {
		const id = $page.params.id;
		if (!id) return;
		loading = true;
		(async () => {
			result = await getResult(id);
			loading = false;
		})();
	});
</script>

{#if loading}
	<p class="opacity-70">{i18n.m.result.loading}</p>
{:else if !result}
	<p>
		{i18n.m.result.notFound}
		<a href={resolve('/runners')} class="text-primary-500 hover:underline"
			>{i18n.m.result.backToRunners}</a
		>
	</p>
{:else}
	<section class="space-y-6">
		<a
			href={resolve('/runners/[id]', { id: result.runner.id })}
			class="inline-flex items-center gap-1 text-sm opacity-70 hover:opacity-100 transition-opacity"
		>
			<ArrowLeft class="size-4" />
			{result.runner.name}
		</a>

		<Hero>
			{#snippet background()}
				<RouteLine
					class="absolute inset-x-6 sm:inset-x-10 top-1/2 -translate-y-1/2 h-12 w-[calc(100%-3rem)] sm:w-[calc(100%-5rem)] text-secondary-500 dark:text-secondary-400 opacity-50"
				/>
			{/snippet}
			<div class="max-w-2xl">
				<h1
					class="text-3xl sm:text-4xl font-semibold leading-tight tracking-tight"
					style="font-family: var(--heading-font-family);"
				>
					{formatRaceName(result.race)}
				</h1>
				<div class="text-sm opacity-85 mt-2">
					{result.event.name} · {formatDate(result.event.date)}{#if result.event.location} · {result.event.location}{/if}
				</div>
			</div>
		</Hero>

		<section class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
			<StatCard
				icon={Route}
				label={i18n.m.result.distance}
				value={formatDistance(result.race.distanceMeters)}
				tone="success"
			/>
			<StatCard
				icon={Clock}
				label={i18n.m.result.finish}
				value={formatTime(result.finishSeconds)}
				tone="primary"
			/>
			<StatCard
				icon={Activity}
				label={i18n.m.result.pace}
				value={formatPace(result.race.distanceMeters, result.finishSeconds)}
				tone="secondary"
			/>
			<StatCard
				icon={Medal}
				label={i18n.m.result.placement}
				value={result.placementOverall ? `#${result.placementOverall}` : '—'}
				sublabel={result.placementCategory && result.category.ageGroup
					? i18n.m.result.placementCategory(result.placementCategory, formatCategory(result.category))
					: undefined}
				tone="tertiary"
			/>
		</section>

		<div class="text-sm opacity-80 flex flex-wrap gap-x-4 gap-y-1">
			<span>
				{i18n.m.result.bib}: <span class="font-mono">{result.bib}</span>
			</span>
			<span>
				{i18n.m.result.category}: {formatCategory(result.category) || '—'}
			</span>
		</div>

		<section>
			<SectionHeader title={i18n.m.result.splitsHeading} icon={Activity} tone="primary" />
			<SplitsTable splits={result.splits ?? []} />
		</section>

		{#if result.conditions}
			<section>
				<SectionHeader title={i18n.m.result.conditionsHeading} icon={Sun} tone="tertiary" />
				<p class="opacity-90">{result.conditions}</p>
			</section>
		{/if}

		{#if result.notes}
			<section>
				<div class="flex items-center gap-3 mb-3">
					<PineSprig class="size-8 text-success-700 dark:text-success-300 shrink-0" />
					<h2
						class="text-lg font-semibold"
						style="font-family: var(--heading-font-family);"
					>
						{i18n.m.result.notesHeading}
					</h2>
				</div>
				<p class="opacity-90 whitespace-pre-wrap">{result.notes}</p>
			</section>
		{/if}
	</section>
{/if}
