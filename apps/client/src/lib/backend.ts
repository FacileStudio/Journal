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

export type LogCursor = {
	ts: string;
	id: number;
};

export type ListLogsResponse = {
	entries: LogEntry[];
	next_before: LogCursor | null;
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
	request_id?: string;
	since?: string;
	until?: string;
	limit?: number;
	before?: LogCursor;
};

export type HistogramParams = {
	app?: string;
	level?: LogLevel[];
	q?: string;
	request_id?: string;
	since?: string;
	until?: string;
};

export type HistogramCounts = {
	debug?: number;
	info?: number;
	warn?: number;
	error?: number;
};

export type HistogramBucket = {
	ts: string;
	counts: HistogramCounts;
};

export type HistogramResponse = {
	bucket_seconds: number;
	buckets: HistogramBucket[];
};

export type LogContextResponse = {
	entries: LogEntry[];
	anchor_id: number;
};

export type ApiKey = {
	id: number;
	app: string;
	prefix: string;
	created_at: string;
	revoked_at: string | null;
};

export type ListApiKeysResponse = {
	keys: ApiKey[];
};

export type CreateApiKeyResponse = {
	key: ApiKey;
	token: string;
};

export type SavedQueryParams = {
	app?: string;
	levels?: string[];
	q?: string;
	request_id?: string;
};

export type SavedQuery = {
	id: number;
	name: string;
	params: SavedQueryParams;
	created_at: string;
};

export type ListSavedQueriesResponse = {
	queries: SavedQuery[];
};

export type SavedQueryResponse = {
	query: SavedQuery;
};

export type AlertRule = {
	id: number;
	name: string;
	saved_query_id: number;
	query_name: string;
	threshold: number;
	window_minutes: number;
	webhook_url: string;
	webhook_header: string;
	enabled: boolean;
	last_fired_at: string | null;
	created_at: string;
};

export type ListAlertsResponse = {
	alerts: AlertRule[];
};

export type AlertResponse = {
	alert: AlertRule;
};

export type CreateAlertParams = {
	name: string;
	saved_query_id: number;
	threshold: number;
	window_minutes: number;
	webhook_url: string;
	webhook_header?: string;
	webhook_secret?: string;
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
		if (browser) {
			goto('/login');
			return new Promise<never>(() => {});
		}
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
		if (params.request_id) qs.set('request_id', params.request_id);
		if (params.since) qs.set('since', params.since);
		if (params.until) qs.set('until', params.until);
		if (params.limit != null) qs.set('limit', String(params.limit));
		if (params.before != null) {
			qs.set('before_ts', params.before.ts);
			qs.set('before_id', String(params.before.id));
		}
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<ListLogsResponse>(`/logs${query}`);
	},

	histogram(params: HistogramParams = {}) {
		const qs = new URLSearchParams();
		if (params.app) qs.set('app', params.app);
		if (params.level?.length) qs.set('level', params.level.join(','));
		if (params.q) qs.set('q', params.q);
		if (params.request_id) qs.set('request_id', params.request_id);
		if (params.since) qs.set('since', params.since);
		if (params.until) qs.set('until', params.until);
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<HistogramResponse>(`/logs/histogram${query}`);
	},

	logContext(id: number, before = 50, after = 50) {
		return apiFetch<LogContextResponse>(`/logs/${id}/context?before=${before}&after=${after}`);
	},

	listApps() {
		return apiFetch<ListAppsResponse>('/apps');
	},

	listApiKeys() {
		return apiFetch<ListApiKeysResponse>('/apikeys');
	},

	createApiKey(app: string) {
		return apiFetch<CreateApiKeyResponse>('/apikeys', {
			method: 'POST',
			body: JSON.stringify({ app })
		});
	},

	revokeApiKey(id: number) {
		return apiFetch<Record<string, never>>(`/apikeys/${id}`, { method: 'DELETE' });
	},

	listQueries() {
		return apiFetch<ListSavedQueriesResponse>('/queries');
	},

	createQuery(name: string, params: SavedQueryParams) {
		return apiFetch<SavedQueryResponse>('/queries', {
			method: 'POST',
			body: JSON.stringify({ name, params })
		});
	},

	deleteQuery(id: number) {
		return apiFetch<Record<string, never>>(`/queries/${id}`, { method: 'DELETE' });
	},

	listAlerts() {
		return apiFetch<ListAlertsResponse>('/alerts');
	},

	createAlert(params: CreateAlertParams) {
		return apiFetch<AlertResponse>('/alerts', {
			method: 'POST',
			body: JSON.stringify(params)
		});
	},

	updateAlert(id: number, params: { enabled: boolean }) {
		return apiFetch<AlertResponse>(`/alerts/${id}`, {
			method: 'PATCH',
			body: JSON.stringify(params)
		});
	},

	deleteAlert(id: number) {
		return apiFetch<Record<string, never>>(`/alerts/${id}`, { method: 'DELETE' });
	}
};
