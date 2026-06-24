<script lang="ts">
	import { resolve } from '$app/paths';
	import Calendar from '@lucide/svelte/icons/calendar';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import type { RaceEvent } from '$lib/types';
	import { formatDate } from '$lib/format';

	let { event, tone = 'primary' }: { event: RaceEvent; tone?: 'primary' | 'secondary' } = $props();

	const stripe = $derived(tone === 'primary' ? 'bg-primary-500' : 'bg-secondary-500');
</script>

<a
	href={resolve('/events/[id]', { id: event.id })}
	class="group relative card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden block transition hover:-translate-y-0.5 hover:shadow-lg"
>
	<div class="h-1.5 w-full {stripe}" aria-hidden="true"></div>
	<div class="p-5 flex items-start gap-4">
		<div class="flex-1 min-w-0">
			<div class="font-heading text-lg font-semibold leading-snug" style="font-family: var(--heading-font-family);">
				{event.name}
			</div>
			<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm opacity-80 mt-2">
				<span class="flex items-center gap-1.5">
					<Calendar class="size-4 opacity-70" />
					{formatDate(event.date)}
				</span>
				{#if event.location}
					<span class="flex items-center gap-1.5">
						<MapPin class="size-4 opacity-70" />
						{event.location}
					</span>
				{/if}
			</div>
		</div>
		<ChevronRight
			class="size-5 shrink-0 mt-1 opacity-40 transition-transform group-hover:translate-x-0.5 group-hover:opacity-100"
		/>
	</div>
</a>
