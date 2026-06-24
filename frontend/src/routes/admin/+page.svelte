<script lang="ts">
	import { onMount } from 'svelte';
	import Users from '@lucide/svelte/icons/users';
	import Calendar from '@lucide/svelte/icons/calendar';
	import Footprints from '@lucide/svelte/icons/footprints';
	import Trophy from '@lucide/svelte/icons/trophy';
	import UserCog from '@lucide/svelte/icons/user-cog';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import { listInvites, listUsers, type AdminInvite } from '$lib/admin/api';
	import type { AdminUser } from '$lib/admin/auth.svelte';
	import StatCard from '$lib/components/StatCard.svelte';

	let users: AdminUser[] = $state([]);
	let invites: AdminInvite[] = $state([]);

	onMount(async () => {
		users = await listUsers();
		invites = await listInvites();
	});

	const pendingInvites = $derived(invites.filter((i) => !i.acceptedAt));
</script>

<section class="space-y-6">
	<header>
		<h1
			class="text-2xl font-semibold leading-tight"
			style="font-family: var(--heading-font-family);"
		>
			{i18n.m.admin.dashboard.heading}
		</h1>
		<p class="text-sm opacity-80 mt-1">{i18n.m.admin.dashboard.subheading}</p>
	</header>

	<section class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
		<StatCard icon={Users} label={i18n.m.admin.dashboard.runners} value={dataStore.runners.length} tone="primary" />
		<StatCard icon={Calendar} label={i18n.m.admin.dashboard.events} value={dataStore.events.length} tone="secondary" />
		<StatCard icon={Footprints} label={i18n.m.admin.dashboard.races} value={dataStore.races.length} tone="tertiary" />
		<StatCard icon={Trophy} label={i18n.m.admin.dashboard.results} value={dataStore.results.length} tone="success" />
		<StatCard
			icon={UserCog}
			label={i18n.m.admin.dashboard.users}
			value={users.length}
			sublabel={pendingInvites.length > 0 ? i18n.m.admin.dashboard.pendingInvites(pendingInvites.length) : undefined}
			tone="primary"
		/>
	</section>
</section>
