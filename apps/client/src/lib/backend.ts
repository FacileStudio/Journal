import { createApi, redirectToLogin } from '@facile/client';

const backendBaseUrl = '/api';

/* The suite's fetch kernel: same-origin cookie, X-Facile-CSRF on every write, the tronc error
   envelope as ApiError, and a 401 off the auth routes sends the browser to /login. The auth
   routes call it quiet, because a 401 there is the answer the caller asked for. */
const apiFetch = createApi({ base: backendBaseUrl, onUnauthorized: redirectToLogin });

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export type AuthUser = {
	id: number;
	email: string;
	name: string;
	is_admin: boolean;
	avatar_url: string;
	locale: string;
	created_at: string;
};

export type MeResponse = {
	user: AuthUser;
};

export type InvitationsAnswer = {
	count: number;
	href?: string;
};

export type AuthConfig = {
	sso_only: boolean;
	oidc_enabled: boolean;
};

// Where the browser goes to start single sign-on. It is a navigation, not a
// fetch: the API answers with a redirect to the identity provider, and an XHR
// cannot follow that across origins.
export const ssoLoginUrl = `${backendBaseUrl}/auth/oidc`;

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
	source?: string;
	request_id?: string;
	session_id?: string;
	since?: string;
	until?: string;
	limit?: number;
	before?: LogCursor;
};

export type HistogramParams = {
	app?: string;
	level?: LogLevel[];
	q?: string;
	source?: string;
	request_id?: string;
	session_id?: string;
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

export type StackFrame = {
	raw: string;
	function?: string;
	file?: string;
	line?: number;
	column?: number;
	resolved: boolean;
	source?: string;
	source_line?: number;
	source_column?: number;
	name?: string;
};

export type ResolvedStack = {
	release: string;
	frames: StackFrame[];
	/* How many frames a source map explained. Zero with a release set means the
	   maps for that build were never uploaded — a different problem from an
	   entry that carries no release at all. */
	resolved: number;
};

export type LogContextResponse = {
	entries: LogEntry[];
	anchor_id: number;
};

export type ApiKeyKind = 'secret' | 'public';

export type ApiKey = {
	id: number;
	app: string;
	kind: ApiKeyKind;
	prefix: string;
	/* Public keys only: the exact origins the browser endpoint accepts. */
	allowed_origins: string[];
	daily_quota: number;
	used_today: number;
	created_at: string;
	revoked_at: string | null;
};

export type NewApiKey = {
	app: string;
	kind: ApiKeyKind;
	allowed_origins?: string[];
	daily_quota?: number;
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
	source?: string;
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

export type AlertProvider = 'webhook' | 'antenne';

export type AlertRule = {
	id: number;
	name: string;
	saved_query_id: number;
	query_name: string;
	provider: AlertProvider;
	threshold: number;
	window_minutes: number;
	webhook_url: string;
	webhook_header: string | null;
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
	provider: AlertProvider;
	threshold: number;
	window_minutes: number;
	webhook_url: string;
	webhook_header?: string;
	webhook_secret?: string;
};

export interface AntenneSettings {
	url: string;
	secret: string;
	enabled: boolean;
}

export interface AntenneSettingsResponse {
	settings: AntenneSettings;
	connected: boolean;
	from_env: boolean;
	connect_error?: string;
}

export interface UpdateAntenneSettings {
	url: string;
	secret: string;
	enabled: boolean;
}

export const backend = {
	baseUrl: backendBaseUrl,

	authConfig() {
		return apiFetch<AuthConfig>('/auth/config', {}, { quiet: true });
	},

	logout() {
		return apiFetch<{ logged_out: boolean }>('/auth/logout', { method: 'POST' }, { quiet: true });
	},

	me() {
		return apiFetch<MeResponse>('/auth/me', {}, { quiet: true });
	},

	invitations() {
		return apiFetch<InvitationsAnswer>('/me/invitations');
	},

	deleteMe() {
		return apiFetch<void>('/auth/me', { method: 'DELETE' }, { quiet: true });
	},

	listLogs(params: ListLogsParams = {}) {
		const qs = new URLSearchParams();
		if (params.app) qs.set('app', params.app);
		if (params.level) for (const level of params.level) qs.append('level', level);
		if (params.q) qs.set('q', params.q);
		if (params.source) qs.set('source', params.source);
		if (params.request_id) qs.set('request_id', params.request_id);
		if (params.session_id) qs.set('session_id', params.session_id);
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
		if (params.source) qs.set('source', params.source);
		if (params.request_id) qs.set('request_id', params.request_id);
		if (params.session_id) qs.set('session_id', params.session_id);
		if (params.since) qs.set('since', params.since);
		if (params.until) qs.set('until', params.until);
		const query = qs.size ? `?${qs}` : '';
		return apiFetch<HistogramResponse>(`/logs/histogram${query}`);
	},

	logStack(id: number) {
		return apiFetch<ResolvedStack>(`/logs/${id}/stack`);
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

	createApiKey(key: NewApiKey) {
		return apiFetch<CreateApiKeyResponse>('/apikeys', {
			method: 'POST',
			body: JSON.stringify(key)
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
	},

	getAntenneSettings() {
		return apiFetch<AntenneSettingsResponse>('/settings/antenne');
	},

	updateAntenneSettings(settings: UpdateAntenneSettings) {
		return apiFetch<AntenneSettingsResponse>('/settings/antenne', {
			method: 'PUT',
			body: JSON.stringify({
				antenne_url: settings.url,
				antenne_secret: settings.secret,
				antenne_enabled: settings.enabled
			})
		});
	}
};
