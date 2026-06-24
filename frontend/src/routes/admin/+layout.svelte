<script lang="ts">
	import { page } from '$app/stores';
	import { auth } from '$lib/admin/auth.svelte';
	import LoginForm from '$lib/admin/components/LoginForm.svelte';
	import AdminShell from '$lib/admin/components/AdminShell.svelte';

	let { children } = $props();

	const isAcceptRoute = $derived($page.url.pathname.startsWith('/admin/accept'));
</script>

{#if isAcceptRoute}
	{@render children()}
{:else if !auth.isAuthed}
	<LoginForm />
{:else}
	<AdminShell>
		{@render children()}
	</AdminShell>
{/if}
