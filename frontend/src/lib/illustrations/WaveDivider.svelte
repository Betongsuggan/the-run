<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 1200 64"
	preserveAspectRatio="none"
	aria-hidden="true"
	xmlns="http://www.w3.org/2000/svg"
>
	<path
		d="M0 32 C 100 12, 200 52, 300 32 S 500 12, 600 32 S 800 52, 900 32 S 1100 12, 1200 32 V 64 H 0 Z"
		fill="currentColor"
		fill-opacity="0.25"
	/>
	<path
		d="M0 40 C 120 24, 220 56, 340 40 S 540 24, 660 40 S 860 56, 980 40 S 1180 24, 1200 40 V 64 H 0 Z"
		fill="currentColor"
		fill-opacity="0.5"
	/>
	<path
		d="M0 50 C 140 36, 240 60, 380 50 S 580 36, 720 50 S 920 60, 1060 50 S 1200 40, 1200 50 V 64 H 0 Z"
		fill="currentColor"
	/>
</svg>
