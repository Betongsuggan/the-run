<script lang="ts">
	import type { Snippet } from 'svelte';
	import X from '@lucide/svelte/icons/x';

	let {
		open,
		title,
		onClose,
		children
	}: { open: boolean; title: string; onClose: () => void; children: Snippet } = $props();

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') onClose();
	}
</script>

<svelte:window onkeydown={onKey} />

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button
			type="button"
			aria-label="Close"
			class="absolute inset-0 bg-surface-950/60 backdrop-blur-sm"
			onclick={onClose}
		></button>
		<div
			role="dialog"
			aria-modal="true"
			aria-label={title}
			class="relative z-10 w-full max-w-xl card preset-filled-surface-50-950 border border-surface-200-800 shadow-xl"
		>
			<header
				class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-5 py-3"
			>
				<h2 class="text-lg font-semibold" style="font-family: var(--heading-font-family);">
					{title}
				</h2>
				<button
					type="button"
					class="btn-icon btn-icon-sm hover:bg-surface-100-900"
					aria-label="Close"
					onclick={onClose}
				>
					<X class="size-4" />
				</button>
			</header>
			<div class="px-5 py-4 max-h-[calc(100vh-12rem)] overflow-y-auto">
				{@render children()}
			</div>
		</div>
	</div>
{/if}
