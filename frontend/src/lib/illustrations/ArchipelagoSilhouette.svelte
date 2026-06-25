<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 1200 120"
	preserveAspectRatio="none"
	aria-hidden="true"
	xmlns="http://www.w3.org/2000/svg"
>
	<!-- Far islands -->
	<g fill="currentColor" fill-opacity="0.35">
		<path
			d="M0 90 Q 80 70 160 88 T 320 90 T 480 86 T 640 92 T 800 84 T 960 92 T 1200 86 V 120 H 0 Z"
		/>
	</g>
	<!-- Mid islands with pine clusters -->
	<g fill="currentColor" fill-opacity="0.65">
		<path
			d="M50 100 Q 120 80 200 96 Q 280 110 360 96 Q 440 82 540 100 Q 620 110 720 92 Q 820 80 920 100 Q 1020 112 1150 94 L 1200 100 V 120 H 0 V 100 Z"
		/>
		<!-- pines -->
		<path d="M180 80 l 6 -16 l 6 16 z M195 84 l 6 -20 l 6 20 z M210 80 l 6 -16 l 6 16 z" />
		<path d="M520 84 l 6 -18 l 6 18 z M540 80 l 6 -22 l 6 22 z M560 86 l 6 -16 l 6 16 z" />
		<path d="M880 82 l 6 -18 l 6 18 z M898 86 l 6 -16 l 6 16 z" />
	</g>
	<!-- Foreground island ridge -->
	<g fill="currentColor">
		<path
			d="M0 110 Q 100 96 220 108 Q 360 122 480 108 Q 600 96 720 110 Q 860 122 980 106 Q 1100 96 1200 108 V 120 H 0 Z"
		/>
	</g>
</svg>
