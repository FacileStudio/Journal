package docs

// Registry inventories every application route the Go binary registers. Paths
// are written relative to the /api mount, which is the reference's declared
// server. The /health and /ready probes are deliberately absent: tronc/health
// mounts them at both the root and /api, so a single path here would misstate
// one of the two, and apiref.Undocumented already exempts them.
//
// Keep it in step with each module's RegisterRoutes: apps/api/apiref_test.go
// fails the build when the router grows a route this does not describe.
var Registry = Response{Modules: []Module{
	authModule,
	ingestModule,
	logsModule,
	apiKeysModule,
	queriesModule,
	alertsModule,
}}

var idParam = []Field{{Name: "id", Type: "int64", Description: "Numeric identifier."}}

var (
	errInvalidBody = Error{Status: 400, Code: "invalid_argument", Description: "the request body is malformed or fails validation"}
	errUnauth      = Error{Status: 401, Code: "unauthenticated", Description: "the session token is missing, invalid, or expired"}
	errNotAdmin    = Error{Status: 403, Code: "permission_denied", Description: "the session belongs to a non-admin user"}
	errSessionRate = Error{Status: 429, Code: "rate_limited", Description: "more than 300 requests per minute from this IP"}
	errBadID       = Error{Status: 400, Code: "invalid_argument", Description: "id is not an integer"}
)

var authModule = Module{
	Name:        "auth",
	Description: "Dashboard accounts, through the suite's porte kit. Passwords are Argon2id; a session is a random 32 byte token stored SHA-256 hashed and valid for 30 days, sent as Authorization: Bearer <token> or, after an SSO login, as an HttpOnly __Host-session cookie. A cookie-authenticated request that is not GET, HEAD or OPTIONS must also carry an X-Facile-CSRF header with any non-empty value.",
	Routes: []Route{
		{
			Method:       "GET",
			Path:         "/auth/config",
			Summary:      "Read the login screen's configuration",
			Description:  "sso_only and oidc_enabled come from porte and follow OIDC_ISSUER and SSO_ONLY. allow_registration is Journal's own addition to the suite-wide shape.",
			ResponseBody: `{"sso_only":bool,"oidc_enabled":bool,"allow_registration":bool}`,
			Errors:       []Error{errSessionRate},
		},
		{
			Method:       "POST",
			Path:         "/auth/register",
			Summary:      "Create an account",
			Description:  "The first account created becomes admin. The user count is taken inside a transaction holding a pg_advisory_xact_lock, so ALLOW_REGISTRATION=false cannot lock an empty instance out. Returns 201.",
			RequestBody:  `{"email":string,"name":string,"password":string}`,
			ResponseBody: `{"token":string,"user":User}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "the email does not parse or the password is under 12 characters"},
				{Status: 403, Code: "permission_denied", Description: "ALLOW_REGISTRATION=false and at least one account exists"},
				{Status: 409, Code: "already_exists", Description: "an account with this email already exists"},
				{Status: 429, Code: "rate_limited", Description: "more than 20 requests per minute from this IP on this endpoint"},
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/login",
			Summary:      "Exchange credentials for a session token",
			Description:  "A wrong password and an unknown account are the same 401. Expired session rows are opportunistically deleted here.",
			RequestBody:  `{"email":string,"password":string}`,
			ResponseBody: `{"token":string,"user":User}`,
			Errors: []Error{
				errInvalidBody,
				{Status: 401, Code: "unauthenticated", Description: "invalid email or password"},
				{Status: 429, Code: "rate_limited", Description: "more than 20 requests per minute from this IP on this endpoint"},
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/logout",
			Summary:      "Delete the current session",
			Description:  "Served by porte. Revokes the one session this request authenticated with, by id, so it cannot end somebody else's; the session cookie is cleared under both its prefixed and unprefixed names.",
			Auth:         "bearer",
			ResponseBody: `{"logged_out":true}`,
			Errors:       []Error{errUnauth, errSessionRate},
		},
		{
			Method:      "GET",
			Path:        "/auth/oidc",
			Summary:     "Start the single sign-on flow",
			Description: "Registered only when OIDC_ISSUER is set. Redirects to the identity provider with PKCE, a nonce and a state, all three carried in one short-lived HttpOnly cookie. Add ?flow=cli (and ?port=N when listening on loopback) to end on a one-time code instead of a cookie.",
			Errors:      []Error{errSessionRate},
		},
		{
			Method:      "GET",
			Path:        "/auth/oidc/callback",
			Summary:     "Finish the single sign-on flow",
			Description: "The provider's redirect target. Verifies the state in constant time, the ID token and its nonce, upserts the account — matched on (issuer, subject), never on an unverified email — sets the session cookie and redirects to OIDC_SUCCESS_URL. No token ever reaches the URL.",
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "the login expired, the state did not match, or the provider returned no email"},
				{Status: 401, Code: "unauthenticated", Description: "the ID token or its nonce did not verify"},
				errSessionRate,
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/oidc/exchange",
			Summary:      "Trade a CLI login code for a session token",
			Description:  "The other half of ?flow=cli. The code is single use, hashed at rest and valid for 60 seconds; a replay is refused and logged. Answers with Cache-Control: no-store.",
			RequestBody:  `{"code":string}`,
			ResponseBody: `{"user_id":string,"token":string}`,
			Errors: []Error{
				errInvalidBody,
				{Status: 401, Code: "unauthenticated", Description: "the code is unknown, expired or already used"},
				errSessionRate,
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/sync-profile",
			Summary:      "Refresh the profile from the identity provider",
			Description:  "Rate limited to one call per user per five minutes; synced is false when the window had not elapsed and nothing was fetched.",
			Auth:         "bearer",
			ResponseBody: `{"synced":bool}`,
			Errors:       []Error{errUnauth, errSessionRate},
		},
		{
			Method:       "POST",
			Path:         "/auth/backchannel-logout",
			Summary:      "Revoke every session for a user, on the provider's instruction",
			Description:  "OpenID Connect Back-Channel Logout 1.0. Called by the identity provider, not by a client, with a form-encoded logout_token. This is the only mechanism by which disabling an account in the provider reaches a session Journal already issued.",
			RequestBody:  `logout_token=<jwt> (application/x-www-form-urlencoded)`,
			ResponseBody: `{"logged_out":true}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "the request is malformed or carries no logout_token"},
				{Status: 401, Code: "unauthenticated", Description: "the logout token did not verify, or carried a nonce, which an ID token replayed here would"},
				errSessionRate,
			},
		},
		{
			Method:       "GET",
			Path:         "/auth/me",
			Summary:      "Read the authenticated user",
			Auth:         "bearer",
			ResponseBody: `{"user":User}`,
			Errors: []Error{
				errUnauth,
				{Status: 404, Code: "not_found", Description: "the session points at a deleted user"},
				errSessionRate,
			},
		},
	},
}

var ingestModule = Module{
	Name:        "ingest",
	Description: "The write path every Facile app ships log entries to. Authenticated with a per-app API key (journal_<app>_…) or the legacy unscoped INGEST_TOKEN.",
	Routes: []Route{
		{
			Method:  "POST",
			Path:    "/ingest",
			Summary: "Ingest one log entry or a batch",
			Description: "Accepts a single entry or {\"entries\":[…]}, at most 1000 per batch. " +
				"Entry fields: app (required with the legacy token; with a per-app key it may be omitted and is filled from the key, or must equal the key's app), " +
				"level (debug|info|warn|error, default info), message (required, truncated at 64 KiB on a rune boundary with \" [truncated]\" appended), " +
				"ts (optional RFC3339 stored as created_at; more than 5 minutes in the future is replaced by server time), and meta (optional object stored as jsonb).\n\n" +
				"**Body size: the 8 MB cap applies to uncompressed bodies too.** Content-Encoding: gzip is accepted, and the two limits are 8 MB on the body as it arrives and 32 MB once decompressed — " +
				"32 MB is only the decompressed ceiling, never a licence to POST a 20 MB plain JSON batch. A plain body above 8 MB is rejected with 413, so a shipper batching large meta payloads must split them.\n\n" +
				"Rate limiting is 600/min keyed on the SHA-256 of the bearer token and answers 429 with Retry-After: 60. " +
				"Shippers should buffer and retry on 429 and 5xx, and drop on other 4xx. Returns 201; entries are written in batches of 500.",
			Auth:         "bearer",
			RequestBody:  `{"app":string,"level":string,"message":string,"ts":string,"meta":object} or {"entries":[…]}`,
			ResponseBody: `{"ingested":int}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "unknown level, missing message, unparseable ts, over 1000 entries, or an app the API key is not scoped to"},
				{Status: 401, Code: "unauthenticated", Description: "no bearer token, or one matching neither an active API key nor INGEST_TOKEN"},
				{Status: 413, Code: "resource_exhausted", Description: "the body exceeds 8 MB, or 32 MB once decompressed"},
				{Status: 429, Code: "rate_limited", Description: "more than 600 requests per minute for this token"},
			},
		},
	},
}

var logsModule = Module{
	Name:        "logs",
	Description: "The read path behind the dashboard: search, histogram, and surrounding-context queries over log_entries.",
	Routes: []Route{
		{
			Method:  "GET",
			Path:    "/logs",
			Summary: "Search log entries",
			Description: "Query parameters: app (exact), level (repeatable or comma-separated), q (full text via websearch_to_tsquery('simple', q)), " +
				"request_id (exact on meta->>'request_id'), since and until (RFC3339 on created_at), limit (default 100, clamped to 1000), " +
				"and the keyset cursor before_ts (RFC3339Nano) plus before_id, which must be given together. " +
				"Ordered created_at desc, id desc; next_before is non-null only when the page came back full.",
			Auth:         "bearer",
			ResponseBody: `{"entries":[LogEntry],"next_before":{"ts":string,"id":int64}|null}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "half a cursor, or a timestamp that is not RFC3339"},
				errUnauth,
				errSessionRate,
			},
		},
		{
			Method:  "GET",
			Path:    "/logs/histogram",
			Summary: "Bucketed entry counts by level",
			Description: "Takes the same filters as /logs minus the cursor and limit. until defaults to now and since to 24 hours before it. " +
				"The server picks the smallest bucket from 1m, 5m, 15m, 1h, 6h, or 1d that yields at most 90 buckets. " +
				"Empty buckets and zero-count levels are omitted — the client fills the gaps.",
			Auth:         "bearer",
			ResponseBody: `{"bucket_seconds":int,"buckets":[{"ts":string,"counts":{level:int}}]}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "until is not after since, or a timestamp is not RFC3339"},
				errUnauth,
				errSessionRate,
			},
		},
		{
			Method:       "GET",
			Path:         "/logs/{id}/context",
			Summary:      "Read the stream around one entry",
			Description:  "Ignores every filter. before and after default to 50 and are capped at 200 each. Ordered newest first with the anchor included.",
			Auth:         "bearer",
			PathParams:   []Field{{Name: "id", Type: "int64", Description: "The anchor log entry."}},
			ResponseBody: `{"entries":[LogEntry],"anchor_id":int64}`,
			Errors: []Error{
				errBadID,
				errUnauth,
				{Status: 404, Code: "not_found", Description: "no log entry with that id"},
				errSessionRate,
			},
		},
		{
			Method:       "GET",
			Path:         "/apps",
			Summary:      "List the apps that have shipped entries",
			Description:  "Grouped by app and ordered by most recently seen. This is what the dashboard's filter rail reads.",
			Auth:         "bearer",
			ResponseBody: `{"apps":[{"name":string,"count":int64,"last_seen":string}]}`,
			Errors:       []Error{errUnauth, errSessionRate},
		},
	},
}

var apiKeysModule = Module{
	Name:        "apikeys",
	Description: "Per-app ingest credentials. Admin only. The row stores a display prefix and the SHA-256 hash; the full token is returned exactly once.",
	Routes: []Route{
		{
			Method:       "GET",
			Path:         "/apikeys",
			Summary:      "List API keys",
			Auth:         "bearer",
			ResponseBody: `{"keys":[{"id":int64,"app":string,"prefix":string,"created_at":string,"revoked_at":string|null}]}`,
			Errors:       []Error{errUnauth, errNotAdmin, errSessionRate},
		},
		{
			Method:  "POST",
			Path:    "/apikeys",
			Summary: "Mint an API key for one app",
			Description: "app must match ^[a-z0-9][a-z0-9-]{0,63}$. Returns 201, and token is the only time the full credential is visible. " +
				"Several active keys per app are allowed, which is what makes zero-downtime rotation possible: mint the new key, redeploy the shipper, then revoke the old one.",
			Auth:         "bearer",
			RequestBody:  `{"app":string}`,
			ResponseBody: `{"key":APIKey,"token":string}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "app does not match ^[a-z0-9][a-z0-9-]{0,63}$"},
				errUnauth,
				errNotAdmin,
				errSessionRate,
			},
		},
		{
			Method:      "DELETE",
			Path:        "/apikeys/{id}",
			Summary:     "Revoke an API key",
			Description: "Sets revoked_at and returns 204. Revoking an already-revoked key is a no-op.",
			Auth:        "bearer",
			PathParams:  idParam,
			Errors: []Error{
				errBadID,
				errUnauth,
				errNotAdmin,
				{Status: 404, Code: "not_found", Description: "no API key with that id"},
				errSessionRate,
			},
		},
	},
}

var queriesModule = Module{
	Name:        "queries",
	Description: "Saved filter sets. params is {app, levels, q, request_id} with no time bounds, since a saved time range would go stale.",
	Routes: []Route{
		{
			Method:       "GET",
			Path:         "/queries",
			Summary:      "List saved queries",
			Description:  "Ordered by name.",
			Auth:         "bearer",
			ResponseBody: `{"queries":[{"id":int64,"name":string,"params":object,"created_at":string}]}`,
			Errors:       []Error{errUnauth, errSessionRate},
		},
		{
			Method:       "POST",
			Path:         "/queries",
			Summary:      "Save a filter set",
			Description:  "Returns 201. levels must be a subset of debug, info, warn, error.",
			Auth:         "bearer",
			RequestBody:  `{"name":string,"params":{"app":string,"levels":[string],"q":string,"request_id":string}}`,
			ResponseBody: `{"query":SavedQuery}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "the name is empty or levels contains an unknown level"},
				errUnauth,
				{Status: 409, Code: "already_exists", Description: "a saved query with this name already exists"},
				errSessionRate,
			},
		},
		{
			Method:      "DELETE",
			Path:        "/queries/{id}",
			Summary:     "Delete a saved query",
			Description: "Returns 204, and is idempotent for an unknown id. A query an alert rule references is refused, checked explicitly and again through the ON DELETE RESTRICT foreign key.",
			Auth:        "bearer",
			PathParams:  idParam,
			Errors: []Error{
				errBadID,
				errUnauth,
				{Status: 409, Code: "already_exists", Description: "delete dependent alert rules first"},
				errSessionRate,
			},
		},
	},
}

var alertsModule = Module{
	Name:        "alerts",
	Description: "Webhook rules over saved queries. Admin only. A 60 second evaluator fires a rule when its query matches at least threshold entries within the last window_minutes, skips rules that fired inside their own window, and writes last_fired_at only on a 2xx webhook response, so a failing endpoint is retried rather than silently marked delivered.",
	Routes: []Route{
		{
			Method:       "GET",
			Path:         "/alerts",
			Summary:      "List alert rules",
			Auth:         "bearer",
			ResponseBody: `{"alerts":[{"id":int64,"name":string,"saved_query_id":int64,"query_name":string,"threshold":int,"window_minutes":int,"webhook_url":string,"webhook_header":string,"enabled":bool,"last_fired_at":string|null,"created_at":string}]}`,
			Errors:       []Error{errUnauth, errNotAdmin, errSessionRate},
		},
		{
			Method:  "POST",
			Path:    "/alerts",
			Summary: "Create an alert rule",
			Description: "Returns 201. threshold must be at least 1, window_minutes between 1 and 1440, and webhook_url a parseable http or https URL that does not literally name a private, loopback, or metadata address. " +
				"webhook_secret is write-only and never returned; when both header fields are set, delivery sends webhook_header: webhook_secret. " +
				"Delivery refuses redirects, and unless the target host is listed in WEBHOOK_ALLOWED_HOSTS the dialer blocks loopback, link-local, multicast, unspecified, and private addresses.",
			Auth:         "bearer",
			RequestBody:  `{"name":string,"saved_query_id":int64,"threshold":int,"window_minutes":int,"webhook_url":string,"webhook_header":string,"webhook_secret":string}`,
			ResponseBody: `{"alert":AlertRule}`,
			Errors: []Error{
				errInvalidBody,
				errUnauth,
				errNotAdmin,
				{Status: 404, Code: "not_found", Description: "no saved query with that saved_query_id"},
				errSessionRate,
			},
		},
		{
			Method:       "PATCH",
			Path:         "/alerts/{id}",
			Summary:      "Enable or disable an alert rule",
			Description:  "enabled is the only mutable field, and omitting it is 400.",
			Auth:         "bearer",
			PathParams:   idParam,
			RequestBody:  `{"enabled":bool}`,
			ResponseBody: `{"alert":AlertRule}`,
			Errors: []Error{
				{Status: 400, Code: "invalid_argument", Description: "id is not an integer, or enabled is missing"},
				errUnauth,
				errNotAdmin,
				{Status: 404, Code: "not_found", Description: "no alert rule with that id"},
				errSessionRate,
			},
		},
		{
			Method:      "DELETE",
			Path:        "/alerts/{id}",
			Summary:     "Delete an alert rule",
			Description: "Returns 204, and is idempotent for an unknown id.",
			Auth:        "bearer",
			PathParams:  idParam,
			Errors:      []Error{errBadID, errUnauth, errNotAdmin, errSessionRate},
		},
	},
}
