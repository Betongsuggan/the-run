<script lang="ts">
	import { resolve } from '$app/paths';
	import Footprints from '@lucide/svelte/icons/footprints';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import type { Runner } from '$lib/types';

	let { runner }: { runner: Runner } = $props();

	const age = $derived(runner.birthYear ? new Date().getFullYear() - runner.birthYear : undefined);

	function hashHue(s: string): number {
		let h = 0;
		for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
		return Math.abs(h) % 360;
	}

	const hue = $derived(hashHue(runner.id || runner.name));
	const initials = $derived(
		runner.name
			.split(' ')
			.map((p) => p[0])
			.slice(0, 2)
			.join('')
	);
</script>

<a
	href={resolve('/runners/[id]', { id: runner.id })}
	class="group card preset-filled-surface-50-950 border border-surface-200-800 p-4 flex items-center gap-4 transition hover:-translate-y-0.5 hover:shadow-md"
>
	<div
		class="size-12 shrink-0 rounded-full flex items-center justify-center font-bold text-lg text-white shadow-sm"
		style="background: oklch(0.62 0.14 {hue});"
		aria-hidden="true"
	>
		{initials}
	</div>
	<div class="min-w-0 flex-1">
		<div class="font-semibold truncate">{runner.name}</div>
		<div class="flex items-center gap-1.5 text-xs opacity-70 mt-0.5">
			<Footprints class="size-3.5" aria-hidden="true" />
			<span>
				{runner.gender}{#if age} · {age} yrs{/if}
			</span>
		</div>
	</div>
	<ChevronRight
		class="size-4 opacity-30 transition-transform group-hover:translate-x-0.5 group-hover:opacity-80"
		aria-hidden="true"
	/>
</a>
