<script lang="ts">
	import { page } from '$app/stores';
	import PolicyViewer from '$lib/policy/PolicyViewer.svelte';
	import { i18n } from '$lib/i18n/state.svelte';

	const kind = $derived($page.params.kind ?? 'privacy');
	const title = $derived(
		i18n.m.policies.kindLabel[kind as 'privacy'] ?? kind
	);
</script>

<svelte:head>
	<title>{title} — Ingmarsöloppet</title>
</svelte:head>

<PolicyViewer {kind} />
