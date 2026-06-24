<script lang="ts">
	import type { Snippet } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
	import Users from '@lucide/svelte/icons/users';
	import Calendar from '@lucide/svelte/icons/calendar';
	import Timer from '@lucide/svelte/icons/timer';
	import UserCog from '@lucide/svelte/icons/user-cog';
	import LogOut from '@lucide/svelte/icons/log-out';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import { i18n } from '$lib/i18n/state.svelte';
	import { auth } from '$lib/admin/auth.svelte';
	import { logout } from '$lib/admin/api';

	let { children }: { children: Snippet } = $props();

	function isActive(href: string, exact: boolean): boolean {
		const path = $page.url.pathname;
		if (exact) return path === href;
		return path === href || path.startsWith(href + '/');
	}

	async function onLogout() {
		await logout();
		await goto(resolve('/admin'));
	}

	function navClass(active: boolean): string {
		return `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
			active
				? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-200 font-semibold'
				: 'hover:bg-surface-100-900 opacity-90 hover:opacity-100'
		}`;
	}
</script>

<div class="grid gap-6 lg:grid-cols-[16rem_minmax(0,1fr)]">
	<aside
		class="lg:sticky lg:top-20 lg:self-start space-y-4 card preset-filled-surface-50-950 border border-surface-200-800 p-4"
	>
		<div class="text-xs uppercase tracking-wide font-semibold opacity-60">{i18n.m.admin.title}</div>
		<nav>
			<ul class="space-y-1">
				<li>
					<a href={resolve('/admin')} class={navClass(isActive(resolve('/admin'), true))}>
						<LayoutDashboard class="size-4" aria-hidden="true" />
						{i18n.m.admin.nav.dashboard}
					</a>
				</li>
				<li>
					<a href={resolve('/admin/runners')} class={navClass(isActive(resolve('/admin/runners'), false))}>
						<Users class="size-4" aria-hidden="true" />
						{i18n.m.admin.nav.runners}
					</a>
				</li>
				<li>
					<a href={resolve('/admin/events')} class={navClass(isActive(resolve('/admin/events'), false))}>
						<Calendar class="size-4" aria-hidden="true" />
						{i18n.m.admin.nav.events}
					</a>
				</li>
				<li>
					<a href={resolve('/admin/results')} class={navClass(isActive(resolve('/admin/results'), false))}>
						<Timer class="size-4" aria-hidden="true" />
						{i18n.m.admin.nav.results}
					</a>
				</li>
				<li>
					<a href={resolve('/admin/users')} class={navClass(isActive(resolve('/admin/users'), false))}>
						<UserCog class="size-4" aria-hidden="true" />
						{i18n.m.admin.nav.users}
					</a>
				</li>
			</ul>
		</nav>

		<div class="border-t border-surface-200-800 pt-3 space-y-2 text-sm">
			{#if auth.user}
				<div class="text-xs opacity-70">{i18n.m.admin.nav.signedInAs(auth.user.email)}</div>
			{/if}
			<a
				href={resolve('/')}
				class="flex items-center gap-2 rounded-lg px-3 py-2 hover:bg-surface-100-900 opacity-80 hover:opacity-100"
			>
				<ExternalLink class="size-4" aria-hidden="true" />
				{i18n.m.admin.nav.viewPublicSite}
			</a>
			<button
				type="button"
				onclick={onLogout}
				class="w-full flex items-center gap-2 rounded-lg px-3 py-2 hover:bg-surface-100-900 opacity-80 hover:opacity-100 text-left"
			>
				<LogOut class="size-4" aria-hidden="true" />
				{i18n.m.admin.nav.logout}
			</button>
		</div>
	</aside>

	<main class="min-w-0">
		{@render children()}
	</main>
</div>
