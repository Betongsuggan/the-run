<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/http';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Policy } from '$lib/types';
	import { formatDate } from '$lib/format';

	let loading = $state(true);
	let error = $state<string | null>(null);
	let policies = $state<Policy[]>([]);

	onMount(async () => {
		try {
			const res = await api<{ policies: Policy[] }>('/policies');
			policies = res.policies ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	});

	// Privacy keeps its canonical /privacy URL; other kinds use /policies/<kind>.
	function hrefFor(kind: string): string {
		if (kind === 'privacy') return '/privacy';
		return `/policies/${encodeURIComponent(kind)}`;
	}

	function titleFor(kind: string): string {
		return i18n.m.policies.kindLabel[kind as 'privacy'] ?? kind;
	}
</script>

<svelte:head>
	<title>{i18n.m.policies.indexTitle}</title>
</svelte:head>

<section class="space-y-6 max-w-3xl mx-auto">
	<header class="space-y-2">
		<h1 class="h1">{i18n.m.policies.indexHeading}</h1>
		<p class="opacity-80">{i18n.m.policies.indexDescription}</p>
	</header>

	{#if loading}
		<p class="opacity-70">{i18n.m.policies.loading}</p>
	{:else if error}
		<p class="text-error-700 dark:text-error-300">{error}</p>
	{:else if policies.length === 0}
		<p class="opacity-80">{i18n.m.policies.empty}</p>
	{:else}
		<ul class="space-y-3">
			{#each policies as policy (policy.id)}
				<li>
					<a
						href={hrefFor(policy.kind)}
						class="card preset-filled-surface-50-950 border border-surface-200-800 p-4 block hover:bg-surface-100-900 transition-colors space-y-2"
					>
						<div class="flex items-baseline justify-between gap-3 flex-wrap">
							<h2 class="text-lg font-semibold">{titleFor(policy.kind)}</h2>
							<span class="text-xs font-mono opacity-70">
								{i18n.m.policies.versionLabel(policy.slug, policy.revision)}
							</span>
						</div>
						<div class="text-xs opacity-70">
							{i18n.m.policies.effectiveFrom(formatDate(policy.effectiveFrom.slice(0, 10)))}
						</div>
						<div class="text-sm text-primary-700 dark:text-primary-300">
							{i18n.m.policies.viewFull} →
						</div>
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</section>
