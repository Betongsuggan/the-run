<script lang="ts">
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded } from '$lib/types';
	import { formatPace, formatDate } from '$lib/format';

	let { results }: { results: ResultExpanded[] } = $props();

	type Point = {
		date: string;
		secondsPerKm: number;
		label: string;
		distanceMeters: number;
	};

	const points: Point[] = $derived(
		results
			.filter((r) => r.race.discipline === 'run')
			.map((r) => ({
				date: r.event.date,
				secondsPerKm: r.finishSeconds / (r.race.distanceMeters / 1000),
				label: `${r.race.name} — ${formatDate(r.event.date)}`,
				distanceMeters: r.race.distanceMeters
			}))
			.sort((a, b) => (a.date < b.date ? -1 : 1))
	);

	const W = 480;
	const H = 200;
	const padLeft = 56;
	const padRight = 16;
	const padTop = 16;
	const padBottom = 32;
	const innerW = W - padLeft - padRight;
	const innerH = H - padTop - padBottom;

	const stats = $derived.by(() => {
		if (points.length === 0) {
			return { min: 0, max: 0, range: 0 };
		}
		const vals = points.map((p) => p.secondsPerKm);
		const min = Math.min(...vals);
		const max = Math.max(...vals);
		const range = max - min || 60;
		return { min: min - range * 0.1, max: max + range * 0.1, range: range * 1.2 };
	});

	const xFor = (i: number) =>
		padLeft + (points.length <= 1 ? innerW / 2 : (i / (points.length - 1)) * innerW);
	const yFor = (sec: number) => padTop + innerH - ((sec - stats.min) / stats.range) * innerH;

	const pathD = $derived(
		points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${xFor(i)} ${yFor(p.secondsPerKm)}`).join(' ')
	);

	const yTicks = $derived.by(() => {
		if (stats.range === 0) return [] as number[];
		const step = stats.range / 4;
		return [0, 1, 2, 3, 4].map((k) => stats.min + step * k);
	});

	function formatPaceFromSec(sec: number): string {
		const m = Math.floor(sec / 60);
		const s = Math.round(sec % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

{#if points.length === 0}
	<p class="text-sm opacity-70">{i18n.m.chart.noRunsYet}</p>
{:else}
	<svg
		viewBox={`0 0 ${W} ${H}`}
		class="w-full h-auto"
		role="img"
		aria-label={i18n.m.runner.paceOverTime}
	>
		{#each yTicks as t (t)}
			<line
				x1={padLeft}
				x2={W - padRight}
				y1={yFor(t)}
				y2={yFor(t)}
				class="stroke-surface-300-700"
				stroke-width="0.5"
			/>
			<text
				x={padLeft - 6}
				y={yFor(t)}
				text-anchor="end"
				dominant-baseline="middle"
				class="fill-current text-[10px] opacity-70"
			>
				{formatPaceFromSec(t)}
			</text>
		{/each}

		<path d={pathD} class="stroke-primary-500" stroke-width="2" fill="none" />

		{#each points as p, i (i)}
			<circle cx={xFor(i)} cy={yFor(p.secondsPerKm)} r="4" class="fill-primary-500">
				<title
					>{p.label} — {formatPace(
						p.distanceMeters,
						p.secondsPerKm * (p.distanceMeters / 1000)
					)}</title
				>
			</circle>
			<text x={xFor(i)} y={H - 12} text-anchor="middle" class="fill-current text-[10px] opacity-70">
				{p.date.slice(0, 4)}
			</text>
		{/each}
	</svg>
	<p class="text-xs opacity-60 mt-2">{i18n.m.chart.paceHelp}</p>
{/if}
