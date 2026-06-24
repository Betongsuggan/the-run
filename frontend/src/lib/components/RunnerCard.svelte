<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Runner } from '$lib/types';

	let { runner }: { runner: Runner } = $props();

	const age = $derived(runner.birthYear ? new Date().getFullYear() - runner.birthYear : undefined);
</script>

<a
	href={resolve('/runners/[id]', { id: runner.id })}
	class="card preset-filled-surface-100-900 border-[1px] border-surface-200-800 p-4 flex items-center gap-4 hover:preset-tonal-primary transition"
>
	<div
		class="size-12 rounded-full preset-filled-primary-500 flex items-center justify-center font-bold text-lg"
	>
		{runner.name
			.split(' ')
			.map((p) => p[0])
			.slice(0, 2)
			.join('')}
	</div>
	<div>
		<div class="font-semibold">{runner.name}</div>
		<div class="text-xs opacity-70">
			{runner.gender}{#if age}
				· {age} yrs
			{/if}
		</div>
	</div>
</a>
