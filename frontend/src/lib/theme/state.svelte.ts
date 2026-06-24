export type ThemeMode = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

const STORAGE_KEY = 'the-run.theme';
const DEFAULT_MODE: ThemeMode = 'system';

function readStored(): ThemeMode {
	if (typeof window === 'undefined') return DEFAULT_MODE;
	const v = window.localStorage.getItem(STORAGE_KEY);
	return v === 'light' || v === 'dark' || v === 'system' ? v : DEFAULT_MODE;
}

function systemPrefersDark(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function resolve(mode: ThemeMode): ResolvedTheme {
	if (mode === 'system') return systemPrefersDark() ? 'dark' : 'light';
	return mode;
}

function apply(resolved: ResolvedTheme): void {
	if (typeof document === 'undefined') return;
	document.documentElement.classList.toggle('dark', resolved === 'dark');
}

let _mode = $state<ThemeMode>(readStored());
let _resolved = $state<ResolvedTheme>(resolve(_mode));

if (typeof window !== 'undefined') {
	const mql = window.matchMedia('(prefers-color-scheme: dark)');
	mql.addEventListener('change', () => {
		if (_mode === 'system') {
			_resolved = systemPrefersDark() ? 'dark' : 'light';
			apply(_resolved);
		}
	});
}

export const theme = {
	get mode(): ThemeMode {
		return _mode;
	},
	get resolved(): ResolvedTheme {
		return _resolved;
	},
	set(next: ThemeMode): void {
		_mode = next;
		_resolved = resolve(next);
		apply(_resolved);
		if (typeof window !== 'undefined') {
			window.localStorage.setItem(STORAGE_KEY, next);
		}
	},
	cycle(): void {
		const order: ThemeMode[] = ['light', 'dark', 'system'];
		const idx = order.indexOf(_mode);
		this.set(order[(idx + 1) % order.length]);
	}
};
