import { browser } from '$app/environment';

export type AdminUser = {
	id: string;
	email: string;
	createdAt: string;
};

const TOKEN_KEY = 'the-run.token';
const USER_KEY = 'the-run.user';

function loadToken(): string | null {
	if (!browser) return null;
	return window.localStorage.getItem(TOKEN_KEY);
}

function loadUser(): AdminUser | null {
	if (!browser) return null;
	const raw = window.localStorage.getItem(USER_KEY);
	if (!raw) return null;
	try {
		return JSON.parse(raw) as AdminUser;
	} catch {
		return null;
	}
}

let _token = $state<string | null>(loadToken());
let _user = $state<AdminUser | null>(loadUser());

export const auth = {
	get token(): string | null {
		return _token;
	},
	get user(): AdminUser | null {
		return _user;
	},
	get isAuthed(): boolean {
		return _token !== null && _user !== null;
	},
	setSession(token: string, user: AdminUser): void {
		_token = token;
		_user = user;
		if (browser) {
			window.localStorage.setItem(TOKEN_KEY, token);
			window.localStorage.setItem(USER_KEY, JSON.stringify(user));
		}
	},
	clear(): void {
		_token = null;
		_user = null;
		if (browser) {
			window.localStorage.removeItem(TOKEN_KEY);
			window.localStorage.removeItem(USER_KEY);
		}
	}
};
