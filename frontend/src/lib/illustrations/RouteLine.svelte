<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 800 60"
	preserveAspectRatio="none"
	aria-hidden="true"
	xmlns="http://www.w3.org/2000/svg"
>
	<path
		d="M 12 32 Q 80 12, 160 32 T 320 30 T 480 34 T 640 26 T 788 32"
		fill="none"
		stroke="currentColor"
		stroke-width="2.5"
		stroke-linecap="round"
		stroke-dasharray="2 8"
		opacity="0.7"
	/>
	<!-- Start pin -->
	<circle cx="12" cy="32" r="6" fill="currentColor" />
	<circle
		cx="12"
		cy="32"
		r="2.5"
		fill="currentColor"
		fill-opacity="0"
		stroke="white"
		stroke-width="1.5"
	/>
	<!-- End flag -->
	<g transform="translate(782 18)">
		<rect width="1.5" height="22" fill="currentColor" />
		<path d="M 1.5 0 L 12 4 L 1.5 8 Z" fill="currentColor" />
	</g>
</svg>
