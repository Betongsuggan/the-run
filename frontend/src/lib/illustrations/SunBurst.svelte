<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg class={className} viewBox="0 0 240 240" aria-hidden="true" xmlns="http://www.w3.org/2000/svg">
	<g stroke="currentColor" stroke-linecap="round" stroke-width="3" opacity="0.5">
		{#each Array(12), i (i)}
			{@const angle = (i * 360) / 12}
			<line
				x1="120"
				y1="120"
				x2="120"
				y2="36"
				transform="rotate({angle} 120 120)"
				stroke-dasharray="2 14"
			/>
		{/each}
	</g>
	<circle cx="120" cy="120" r="48" fill="currentColor" opacity="0.18" />
	<circle cx="120" cy="120" r="32" fill="currentColor" opacity="0.32" />
</svg>
