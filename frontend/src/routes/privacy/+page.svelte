<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/http';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Policy, PolicyRevision } from '$lib/types';
	import { renderPolicyMarkdown } from '$lib/policy/render';

	// /privacy reads the currently-published policy by default. When the URL
	// carries ?v=<policyId>&r=<revision> we instead fetch that exact
	// historical revision — used by my-data captions so a runner can read
	// the body they originally consented to.

	let loading = $state(true);
	let error = $state<string | null>(null);
	let current = $state<Policy | null>(null);
	let revision = $state<PolicyRevision | null>(null);

	const wantedPolicyId = $derived($page.url.searchParams.get('v'));
	const wantedRevision = $derived(Number($page.url.searchParams.get('r') ?? '0'));

	onMount(() => {
		void load();
	});

	$effect(() => {
		// React to URL changes so navigating between deep links re-fetches.
		void wantedPolicyId;
		void wantedRevision;
		if (!loading) void load();
	});

	async function load() {
		loading = true;
		error = null;
		try {
			const live = await api<Policy>('/policies/current').catch((err: Error) => {
				// 503 = no published policy. Show the policy-pending state.
				if (/503/.test(err.message)) return null;
				throw err;
			});
			current = live;
			if (wantedPolicyId && wantedRevision > 0) {
				revision = await api<PolicyRevision>(
					`/policies/${encodeURIComponent(wantedPolicyId)}/revisions/${wantedRevision}`
				);
			} else {
				revision = null;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	type Viewing = {
		bodySv: string;
		bodyEn: string;
		revision: number;
		slug: string;
		isHistorical: boolean;
	};

	const viewing: Viewing | null = $derived.by(() => {
		if (revision && current) {
			return {
				bodySv: revision.bodySv,
				bodyEn: revision.bodyEn,
				revision: revision.revision,
				slug: current.slug,
				isHistorical: revision.revision !== current.revision || revision.policyId !== current.id
			};
		}
		if (current) {
			return {
				bodySv: current.bodySv,
				bodyEn: current.bodyEn,
				revision: current.revision,
				slug: current.slug,
				isHistorical: false
			};
		}
		return null;
	});

	const html = $derived.by(() => {
		if (!viewing) return '';
		const body = i18n.locale === 'en' ? viewing.bodyEn || viewing.bodySv : viewing.bodySv;
		return renderPolicyMarkdown(body);
	});
</script>

<svelte:head>
	<title>{i18n.m.privacy.pageTitle}</title>
</svelte:head>

<section class="space-y-6 max-w-3xl mx-auto">
	{#if loading}
		<p class="opacity-70">{i18n.m.privacy.loading}</p>
	{:else if error}
		<p class="text-error-700 dark:text-error-300">{error}</p>
	{:else if !current}
		<header>
			<h1 class="h1">{i18n.m.privacy.heading}</h1>
		</header>
		<p class="opacity-80">{i18n.m.privacy.notPublished}</p>
	{:else if viewing}
		<header class="space-y-2">
			<h1 class="h1">{i18n.m.privacy.heading}</h1>
			<div class="text-sm opacity-70 flex items-center gap-3 flex-wrap">
				<span class="font-mono">{i18n.m.privacy.versionLabel(viewing.slug, viewing.revision)}</span>
				{#if viewing.isHistorical}
					<span class="badge preset-tonal-warning">{i18n.m.privacy.historicalBadge}</span>
					<a href="/privacy" class="text-primary-700 dark:text-primary-300 underline">
						{i18n.m.privacy.viewCurrent}
					</a>
				{/if}
			</div>
		</header>
		<article class="prose dark:prose-invert max-w-none">{@html html}</article>
	{/if}
</section>
