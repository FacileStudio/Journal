const backendBaseUrl = '/api';

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

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
	const response = await fetch(`${backendBaseUrl}${path}`, { ...options, headers });
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
