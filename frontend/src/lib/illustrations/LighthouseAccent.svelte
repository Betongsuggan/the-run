<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 120 160"
	aria-hidden="true"
	xmlns="http://www.w3.org/2000/svg"
>
	<!-- Beam -->
	<g opacity="0.35">
		<path d="M62 40 L 10 8 L 10 24 Z" fill="currentColor" />
		<path d="M62 40 L 10 56 L 10 72 Z" fill="currentColor" opacity="0.6" />
	</g>
	<!-- Tower base -->
	<rect x="48" y="120" width="36" height="22" rx="2" fill="currentColor" opacity="0.6" />
	<!-- Tower body (tapered) -->
	<path d="M52 60 L 80 60 L 84 120 L 48 120 Z" fill="currentColor" opacity="0.9" />
	<!-- Red stripe -->
	<path d="M53.5 84 L 78.5 84 L 79.5 100 L 52.5 100 Z" fill="currentColor" opacity="0.45" />
	<!-- Lamp room -->
	<rect x="50" y="48" width="32" height="14" rx="1.5" fill="currentColor" />
	<!-- Lamp top -->
	<path d="M58 48 L 74 48 L 70 38 L 62 38 Z" fill="currentColor" opacity="0.85" />
	<!-- Spike -->
	<rect x="65" y="28" width="2" height="10" fill="currentColor" />
	<!-- Rocks -->
	<path d="M28 142 Q 60 130 96 142 L 110 152 L 18 152 Z" fill="currentColor" opacity="0.5" />
</svg>
