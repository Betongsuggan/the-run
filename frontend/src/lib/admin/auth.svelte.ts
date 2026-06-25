import { api } from '$lib/http';

export type AdminUser = {
	id: string;
	email: string;
	isAdmin: boolean;
};

export type AuthStep = 'password' | 'totp_verify' | 'totp_enroll' | 'ready';

export type LoginResponseStep = 'verify' | 'enroll';

type LoginResponse = {
	step: LoginResponseStep;
	totpSecret?: string;
	otpauthUri?: string;
};

let _user = $state<AdminUser | null>(null);
let _step = $state<AuthStep>('password');
let _enroll = $state<{ secret: string; otpauthUri: string } | null>(null);
let _hydrated = $state(false);

/**
 * Backend-backed admin auth. There is no token in localStorage — the session
 * lives entirely in an httpOnly cookie set by the API. Call hydrate() on app
 * boot to figure out whether the cookie maps to a valid session.
 */
export const auth = {
	get user(): AdminUser | null {
		return _user;
	},
	get step(): AuthStep {
		return _step;
	},
	get enroll(): { secret: string; otpauthUri: string } | null {
		return _enroll;
	},
	get hydrated(): boolean {
		return _hydrated;
	},
	get isAuthed(): boolean {
		return _user !== null;
	},

	/**
	 * Asks the API who the current session belongs to. Updates state to
	 * 'ready' on success or 'password' on failure. Idempotent.
	 */
	async hydrate(): Promise<void> {
		try {
			const res = await api<{ user: AdminUser }>('/auth/me');
			_user = res.user;
			_step = 'ready';
		} catch {
			_user = null;
			_step = 'password';
		} finally {
			_hydrated = true;
		}
	},

	async login(email: string, password: string): Promise<void> {
		const res = await api<LoginResponse>('/auth/login', {
			method: 'POST',
			body: { email, password }
		});
		if (res.step === 'enroll') {
			_enroll = { secret: res.totpSecret ?? '', otpauthUri: res.otpauthUri ?? '' };
			_step = 'totp_enroll';
		} else {
			_enroll = null;
			_step = 'totp_verify';
		}
	},

	async verifyTotp(code: string): Promise<void> {
		const res = await api<{ user: AdminUser }>('/auth/totp', {
			method: 'POST',
			body: { code }
		});
		_user = res.user;
		_enroll = null;
		_step = 'ready';
	},

	async logout(): Promise<void> {
		try {
			await api<{ ok: boolean }>('/auth/logout', { method: 'POST' });
		} finally {
			_user = null;
			_enroll = null;
			_step = 'password';
		}
	}
};
