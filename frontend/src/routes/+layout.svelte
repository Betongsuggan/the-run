<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import Waves from '@lucide/svelte/icons/waves';
	import { i18n } from '$lib/i18n/state.svelte';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import PineSprig from '$lib/illustrations/PineSprig.svelte';

	let { children } = $props();

	const isHome = $derived($page.url.pathname === '/');
	const isRunners = $derived($page.url.pathname.startsWith('/runners'));
	const isRegister = $derived($page.url.pathname.startsWith('/register'));

	$effect(() => {
		if (typeof document !== 'undefined') {
			document.documentElement.lang = i18n.locale;
		}
	});
</script>

<div class="min-h-screen flex flex-col">
	<header
		class="sticky top-0 z-30 border-b border-surface-200-800 bg-surface-50-950/85 backdrop-blur"
	>
		<nav class="mx-auto max-w-6xl px-4 py-3 flex items-center gap-6">
			<a href={resolve('/')} class="flex items-center gap-2 group">
				<span
					class="inline-flex size-9 items-center justify-center rounded-xl bg-primary-500 text-primary-contrast-500 shadow-sm transition-transform group-hover:scale-105"
					aria-hidden="true"
				>
					<Waves class="size-5" />
				</span>
				<span
					class="font-heading text-xl font-semibold tracking-tight"
					style="font-family: var(--heading-font-family);"
				>
					{i18n.m.nav.brand}
				</span>
			</a>
			<ul class="flex items-center gap-1 text-sm">
				<li>
					<a
						href={resolve('/')}
						class="rounded-full px-3 py-1.5 transition-colors {isHome
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-200 font-semibold'
							: 'opacity-80 hover:opacity-100 hover:bg-surface-100-900'}">{i18n.m.nav.home}</a
					>
				</li>
				<li>
					<a
						href={resolve('/runners')}
						class="rounded-full px-3 py-1.5 transition-colors {isRunners
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-200 font-semibold'
							: 'opacity-80 hover:opacity-100 hover:bg-surface-100-900'}">{i18n.m.nav.runners}</a
					>
				</li>
				<li>
					<a
						href={resolve('/register')}
						class="rounded-full px-3 py-1.5 transition-colors {isRegister
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-200 font-semibold'
							: 'opacity-80 hover:opacity-100 hover:bg-surface-100-900'}">{i18n.m.nav.register}</a
					>
				</li>
			</ul>
			<div class="ml-auto flex items-center gap-2">
				<ThemeToggle />
				<LanguageSwitcher />
			</div>
		</nav>
	</header>

	<main class="flex-1 mx-auto w-full max-w-6xl px-4 py-8">
		{@render children()}
	</main>

	<footer class="border-t border-surface-200-800 py-6">
		<div class="mx-auto max-w-6xl px-4 flex items-center justify-center gap-3 text-xs opacity-70">
			<PineSprig class="size-6 text-success-700 dark:text-success-300" />
			<span>{i18n.m.footer}</span>
			<PineSprig class="size-6 text-success-700 dark:text-success-300 scale-x-[-1]" />
		</div>
	</footer>
</div>
