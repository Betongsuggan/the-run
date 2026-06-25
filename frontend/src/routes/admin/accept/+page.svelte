<script lang="ts">
	import { resolve } from '$app/paths';
	import { i18n } from '$lib/i18n/state.svelte';
</script>

<div class="mx-auto max-w-md py-12">
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 sm:p-8 space-y-4">
		<h1
			class="text-2xl font-semibold leading-tight tracking-tight"
			style="font-family: var(--heading-font-family);"
		>
			{i18n.m.admin.accept.heading}
		</h1>
		<p class="text-sm opacity-80">{i18n.m.admin.accept.iterationNotice}</p>
		<a href={resolve('/admin')} class="btn preset-filled-primary-500 inline-block">
			{i18n.m.admin.accept.backToLogin}
		</a>
	</div>
</div>
