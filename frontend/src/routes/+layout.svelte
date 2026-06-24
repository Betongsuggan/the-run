<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { i18n } from '$lib/i18n/state.svelte';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';

	let { children } = $props();

	const isHome = $derived($page.url.pathname === '/');
	const isRunners = $derived($page.url.pathname.startsWith('/runners'));

	$effect(() => {
		if (typeof document !== 'undefined') {
			document.documentElement.lang = i18n.locale;
		}
	});
</script>

<div class="min-h-screen flex flex-col">
	<header class="border-b border-surface-200-800">
		<nav class="mx-auto max-w-5xl px-4 py-3 flex items-center gap-6">
			<a href={resolve('/')} class="font-bold text-xl">{i18n.m.nav.brand}</a>
			<ul class="flex items-center gap-4 text-sm">
				<li>
					<a
						href={resolve('/')}
						class="hover:underline"
						class:font-semibold={isHome}
						class:text-primary-500={isHome}>{i18n.m.nav.home}</a
					>
				</li>
				<li>
					<a
						href={resolve('/runners')}
						class="hover:underline"
						class:font-semibold={isRunners}
						class:text-primary-500={isRunners}>{i18n.m.nav.runners}</a
					>
				</li>
			</ul>
			<div class="ml-auto">
				<LanguageSwitcher />
			</div>
		</nav>
	</header>

	<main class="flex-1 mx-auto w-full max-w-5xl px-4 py-8">
		{@render children()}
	</main>

	<footer class="border-t border-surface-200-800 py-4 text-center text-xs opacity-70">
		{i18n.m.footer}
	</footer>
</div>
