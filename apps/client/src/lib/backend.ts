import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { clearToken, getToken } from './auth';

const backendBaseUrl = '/api';

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export type AuthUser = {
	id: number;
	email: string;
	name: string;
	is_admin: boolean;
	created_at: string;
};

export type AuthResponse = {
	token: string;
	user: AuthUser;
};

export type MeResponse = {
	user: AuthUser;
};

export type AuthConfig = {
	allow_registration: boolean;
};

export type LogEntry = {
	id: number;
	app: string;
	level: LogLevel;
	message: string;
	meta?: Record<string, unknown>;
	created_at: string;
	received_at: string;
};

export type ListLogsResponse = {
	entries: LogEntry[];
	next_before: number | null;
};

export type AppSummary = {
	name: string;
	count: number;
	last_seen: string;
};

export type ListAppsResponse = {
	apps: AppSummary[];
};

export type ListLogsParams = {
	app?: string;
	level?: LogLevel[];
	q?: string;
	since?: string;
	until?: string;
	limit?: number;
	before?: number;
};

type ApiErrorPayload = {
	error?: { message?: string };
};

async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
	const headers = new Headers(options.headers);
	if (!headers.has('Content-Type') && options.body && !(options.body instanceof FormData)) {
		headers.set('Content-Type', 'application/json');
	}
	const token = getToken();
	if (token) headers.set('Authorization', `Bearer ${token}`);

	const response = await fetch(`${backendBaseUrl}${path}`, { ...options, headers });
	if (response.status === 401 && !path.startsWith('/auth/')) {
		clearToken();
		if (browser) goto('/login');
	}
	if (!response.ok) {
		let payload: ApiErrorPayload | undefined;
		try {
			payload = (await response.json()) as ApiErrorPayload;
		} catch {
			payload = undefined;
		}
		throw new Error(payload?.error?.message || `Request failed with status ${response.status}`);
	}
	const text = await response.text();
	if (!text) return {} as T;
	return JSON.parse(text) as T;
}

export const backend = {
	baseUrl: backendBaseUrl,

	authConfig() {
		return apiFetch<AuthConfig>('/auth/config');
	},

	login(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},

	register(email: string, password: string, name = '') {
		return apiFetch<AuthResponse>('/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, password, name })
		});
	},

	logout() {
		return apiFetch<{ ok: boolean }>('/auth/logout', { method: 'POST' });
	},

	me() {
		return apiFetch<MeResponse>('/auth/me');
	},

	listLogs(params: ListLogsParams = {}) {
		const qs = new URLSearchParams();
		if (params.app) qs.set('app', params.app);
		if (params.level) for (const level of params.level) qs.append('level', level);
		if (params.q) qs.set('q', params.q);
		if (params.since) qs.set('since', params.since);
		if (params.until) qs.set('until', params.until);
		if (params.limit != null) qs.set('limit', String(params.limit));
		if (params.before != null) qs.set('before', String(params.before));
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<ListLogsResponse>(`/logs${query}`);
	},

	listApps() {
		return apiFetch<ListAppsResponse>('/apps');
	}
};
