import { i18n } from '$lib/i18n/state.svelte';
import type { Category, Gender, Race } from '$lib/types';

export function formatTime(totalSeconds: number): string {
	const h = Math.floor(totalSeconds / 3600);
	const m = Math.floor((totalSeconds % 3600) / 60);
	const s = totalSeconds % 60;
	const pad = (n: number) => n.toString().padStart(2, '0');
	return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

export function formatPace(distanceMeters: number, totalSeconds: number): string {
	if (distanceMeters <= 0) return '—';
	const secondsPerKm = totalSeconds / (distanceMeters / 1000);
	const m = Math.floor(secondsPerKm / 60);
	const s = Math.round(secondsPerKm % 60);
	return `${m}:${s.toString().padStart(2, '0')}/km`;
}

export function formatDistance(meters: number): string {
	if (meters < 1000) return `${meters} m`;
	const km = meters / 1000;
	return Number.isInteger(km) ? `${km}K` : `${km.toFixed(2)} km`;
}

export function formatDate(iso: string): string {
	const tag = i18n.locale === 'sv' ? 'sv-SE' : 'en-GB';
	return new Date(iso).toLocaleDateString(tag);
}

export function formatRaceName(race: Race): string {
	if (race.discipline === 'kids') return i18n.m.race.kids;
	const km = race.distanceMeters / 1000;
	const kmStr = Number.isInteger(km) ? `${km}km` : `${km.toFixed(1)}km`;
	return `${kmStr} ${i18n.m.discipline[race.discipline]}`;
}

export function formatGender(g: Gender): string {
	return i18n.m.category.gender[g];
}

export function formatCategory(cat: Category): string {
	const parts: string[] = [];
	if (cat.gender) parts.push(formatGender(cat.gender));
	if (cat.ageGroup) {
		const stripped = cat.ageGroup.replace(/^[MFX](?=\d)/, '');
		parts.push(stripped);
	}
	return parts.join(' ');
}
