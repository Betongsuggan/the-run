<script lang="ts">
	import { resolve } from '$app/paths';
	import Users from '@lucide/svelte/icons/users';
	import Calendar from '@lucide/svelte/icons/calendar';
	import Footprints from '@lucide/svelte/icons/footprints';
	import Trophy from '@lucide/svelte/icons/trophy';
	import UserCog from '@lucide/svelte/icons/user-cog';
	import { i18n } from '$lib/i18n/state.svelte';
	import { dataStore } from '$lib/store/data.svelte';
	import { auth } from '$lib/admin/auth.svelte';
	import StatCard from '$lib/components/StatCard.svelte';

	// User management (multi-admin invites, listing) moved out of the dashboard
	// while we wait for a server-backed implementation in a later slice. The
	// current session lives in auth.user.
	const finishedCount = $derived(
		dataStore.registrations.filter((r) => r.status === 'finished').length
	);
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
		<StatCard
			icon={Users}
			label={i18n.m.admin.dashboard.runners}
			value={dataStore.runners.length}
			tone="primary"
		/>
		<StatCard
			icon={Calendar}
			label={i18n.m.admin.dashboard.events}
			value={dataStore.events.length}
			tone="secondary"
		/>
		<StatCard
			icon={Footprints}
			label={i18n.m.admin.dashboard.races}
			value={dataStore.races.length}
			tone="tertiary"
		/>
		<a
			href={resolve('/admin/results')}
			class="block rounded-2xl hover:-translate-y-0.5 transition-transform"
		>
			<StatCard
				icon={Trophy}
				label={i18n.m.admin.dashboard.results}
				value={finishedCount}
				tone="success"
			/>
		</a>
		<StatCard
			icon={UserCog}
			label={i18n.m.admin.dashboard.users}
			value={auth.user ? 1 : 0}
			tone="primary"
		/>
	</section>
</section>
