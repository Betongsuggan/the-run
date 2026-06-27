<script lang="ts">
	import PolicyViewer from '$lib/policy/PolicyViewer.svelte';
	import { i18n } from '$lib/i18n/state.svelte';
</script>

<svelte:head>
	<title>{i18n.m.policies.pageTitlePrivacy}</title>
</svelte:head>

<PolicyViewer kind="privacy" />
