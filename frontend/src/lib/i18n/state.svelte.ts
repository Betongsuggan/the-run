import { messages, type Locale } from './messages';

const STORAGE_KEY = 'the-run.locale';
const DEFAULT_LOCALE: Locale = 'sv';

function readInitialLocale(): Locale {
	if (typeof window === 'undefined') return DEFAULT_LOCALE;
	const stored = window.localStorage.getItem(STORAGE_KEY);
	return stored === 'sv' || stored === 'en' ? stored : DEFAULT_LOCALE;
}

let _locale = $state<Locale>(readInitialLocale());

export const i18n = {
	get locale(): Locale {
		return _locale;
	},
	get m() {
		return messages[_locale];
	},
	set(next: Locale): void {
		_locale = next;
		if (typeof window !== 'undefined') {
			window.localStorage.setItem(STORAGE_KEY, next);
			document.documentElement.lang = next;
		}
	}
};
