<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 40 80"
	aria-hidden="true"
	xmlns="http://www.w3.org/2000/svg"
>
	<g fill="currentColor">
		<!-- Stem -->
		<rect x="19" y="10" width="2" height="62" rx="1" opacity="0.55" />
		<!-- Needles -->
		<g opacity="0.85">
			<path d="M20 14 l -12 6 l 12 -2 z" />
			<path d="M20 14 l 12 6 l -12 -2 z" />
			<path d="M20 26 l -14 7 l 14 -2 z" />
			<path d="M20 26 l 14 7 l -14 -2 z" />
			<path d="M20 38 l -16 8 l 16 -2 z" />
			<path d="M20 38 l 16 8 l -16 -2 z" />
			<path d="M20 52 l -14 7 l 14 -2 z" />
			<path d="M20 52 l 14 7 l -14 -2 z" />
		</g>
	</g>
</svg>
