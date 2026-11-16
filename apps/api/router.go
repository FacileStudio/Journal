package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/middleware"
	"github.com/FacileStudio/Journal/apps/api/internal/plans"
	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/modules/alerts"
	"github.com/FacileStudio/Journal/apps/api/modules/antenne"
	"github.com/FacileStudio/Journal/apps/api/modules/apikeys"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/modules/deletions"
	"github.com/FacileStudio/Journal/apps/api/modules/ingest"
	"github.com/FacileStudio/Journal/apps/api/modules/logs"
	"github.com/FacileStudio/Journal/apps/api/modules/mcp"
	"github.com/FacileStudio/Journal/apps/api/modules/queries"
	"github.com/FacileStudio/Journal/apps/api/modules/sourcemaps"
	"github.com/FacileStudio/Journal/apps/api/modules/workspaces"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/httpx"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// buildRouter assembles the HTTP tree exactly as run() serves it. Two buckets
// guard the browser endpoint: the per (key, IP) bucket is small — a page that
// manages 60 requests in a minute is a render loop, which is exactly the
// traffic this refuses — and since tronc v0.12.0 its IP really is the visitor's
// rather than Traefik's or Cloudflare's. The per-key ceiling and the daily
// quota are kept anyway: a bound that owes nothing to the network layer is
// worth having even when the network layer is correct, and it is what actually
// holds when a public key leaks.
//
// TrustedProxies decides what RemoteAddr is by the time the rate limiters see
// it, so it is a security setting and not a logging one — unset it defaults to
// loopback plus the private ranges, which is Traefik. CDNProxies and CDNHeader
// ride along from the same variable; they are not redundant with the trust set
// because Traefik replaces the incoming X-Forwarded-For instead of extending
// it, so behind Cloudflare the chain this app sees holds only the edge and the
// visitor survives nowhere but Cf-Connecting-Ip.
//
// A nil mcpServer leaves /mcp and the confirm page routes unmounted. The zero
// tenancy routes no app and reads every workspace's plan as the free one.
func buildRouter(db *gorm.DB, oidcKit *oidc.Kit, sessions *session.Manager, users *kit.Store, appEnv env.Config, appLogger *slog.Logger, antenneService *antenne.Service, sso ssoRoutes, mcpServer *mcpSetup, tenants tenancy, checks ...health.Check) (chi.Router, error) {
	tenants = tenants.orDefaults(db, appEnv, appLogger)
	workspaceService := workspace.NewService(db)
	ingestService := ingest.NewService(db)
	ingestService.Use(tenants.routes, tenants.plans)
	logsService := logs.NewService(db)
	authService := auth.NewService(db)
	apiKeysService := apikeys.NewService(db)
	queriesService := queries.NewService(db)
	sourceMapsService := sourcemaps.NewService(db)
	alertsService := alerts.NewService(db)
	mcpService, err := mcpServer.service(mcp.Deps{
		Logs: logsService, Workspaces: workspaceService, Stacks: sourceMapsService, Queries: queriesService, Alerts: alertsService, Accounts: users,
	})
	if err != nil {
		return nil, err
	}

	sessionLimiter := perMinute(300, troncmiddleware.RemoteAddrKey64)
	ingestLimiter := perMinute(600, middleware.KeyByBearerTokenHash)
	browserIngestLimiter := perMinute(60, middleware.KeyByBrowserKeyAndIP)
	browserKeyCeiling := perMinute(600, middleware.KeyByBrowserKey)

	clientDir := spa.DirFromEnv()
	router := httpx.NewRouter(httpx.Config{
		Logger: appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins: appEnv.CORSAllowedOrigins,
		},
		TrustedProxies: appEnv.TrustedProxies,
		CDNProxies:     appEnv.CDNProxies,
		CDNHeader:      appEnv.CDNHeader,
		Security:       securityConfig(clientDir, appLogger),
	})

	health.Mount(router, checks...)
	router.Get("/version", health.Version(version))
	apiref.Mount(router, referenceConfig())
	if mcpService != nil {
		mcp.RegisterRoutes(router, mcpService)
	}

	requireAuth := kit.RequireAuth(sessions, users)

	router.Route("/api", func(api chi.Router) {
		_ = kit.Mount(api.With(sessionLimiter), kit.Config{Sessions: sessions, OIDC: oidcKit})
		auth.RegisterRoutes(api, authService, sessionLimiter, requireAuth,
			func(w http.ResponseWriter, r *http.Request) { sessions.ClearCookie(w, r, porte.SessionCookieName) })
		sso.register(api.With(sessionLimiter, requireAuth), api, appLogger)
		ingest.RegisterRoutes(api, ingestService, ingestLimiter, middleware.RequireIngestAuth(appEnv.IngestToken, apiKeysService))
		ingest.RegisterBrowserRoutes(api, ingestService, middleware.RequireBrowserIngestAuth(apiKeysService), browserKeyCeiling, browserIngestLimiter)
		sourcemaps.RegisterUploadRoutes(api, sourceMapsService, ingestLimiter, middleware.RequireIngestAuth(appEnv.IngestToken, apiKeysService))

		if mcpService != nil {
			mcp.RegisterPendingRoutes(api, mcpService, sessionLimiter, requireAuth)
		}
		api.Group(func(protected chi.Router) {
			protected.Use(sessionLimiter)
			protected.Use(requireAuth)
			protected.Use(workspace.Middleware(workspaceService))
			workspaces.RegisterRoutes(protected)
			logs.RegisterRoutes(protected, logsService, sourceMapsService)
			queries.RegisterRoutes(protected, queriesService)
			apikeys.RegisterRoutes(protected, apiKeysService)
			alerts.RegisterRoutes(protected, alertsService)
			sourcemaps.RegisterAdminRoutes(protected, sourceMapsService)
			protected.Group(func(admin chi.Router) {
				admin.Use(middleware.RequireAdmin)
				antenne.RegisterRoutes(admin, antenneService, appLogger)
			})
		})
	})

	if spa.Available(clientDir) {
		router.Handle("/*", spa.Handler(spa.Config{Dir: clientDir}))
		appLogger.Info("serving client", slog.String("dir", clientDir))
	}

	return router, nil
}

// ssoRoutes are the routes that exist only with an OIDC issuer: Registre's
// signed deletion push, and the user menu's pending space invitations.
type ssoRoutes struct {
	erasures    *directory.Deletions
	invitations *kit.InvitationsConfig
}

// register mounts the deletion push on the bare API router, since Registre
// signs it and holds no session, and the invitations behind the session.
func (sso ssoRoutes) register(signedIn, api chi.Router, appLogger *slog.Logger) {
	if sso.erasures != nil {
		deletions.RegisterRoutes(api, sso.erasures.Handler())
	}
	if sso.invitations == nil {
		return
	}
	if err := kit.MountInvitations(signedIn, *sso.invitations); err != nil {
		appLogger.Error("failed to mount the invitations route", slog.Any("error", err))
	}
}

// perMinute is a throughput limiter of n requests a minute per key: a token
// bucket refilled at n a minute with room for n, as tronc's
// docs/ratelimit.md prescribes for Journal. A refusal is the 429 envelope the
// SDKs and the dashboard parse, with Retry-After set to the real wait. At most
// 50,000 keys are remembered, so an address-rotating caller cannot grow the
// map without bound.
func perMinute(n int, key func(*http.Request) string) func(http.Handler) http.Handler {
	return troncmiddleware.Ratelimit(troncmiddleware.RatelimitConfig{
		Limit: float64(n) / 60, Burst: n, KeyFunc: key, MaxKeys: 50000, OnRefuse: refuseRateLimited,
	})
}

func refuseRateLimited(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteError(w, errors.RateLimited("rate limit exceeded"))
}

// tenancy is how ingest places entries in workspaces and how their plans are
// read: routes from INGEST_ROUTES, and the plan service fed by Registre.
type tenancy struct {
	routes workspace.Router
	plans  *plans.Service
}

func (t tenancy) orDefaults(db *gorm.DB, appEnv env.Config, logger *slog.Logger) tenancy {
	if t.plans == nil {
		t.plans = plans.New(db, appEnv.Porte.Issuer, logger)
	}
	return t
}

// startTenancy creates the workspaces INGEST_ROUTES names and starts pulling
// the plans from Registre's directory snapshot when an issuer is configured.
func startTenancy(ctx context.Context, db *gorm.DB, appEnv env.Config, registre directory.Client, logger *slog.Logger) (tenancy, error) {
	routes, err := workspace.NewService(db).SyncRoutes(ctx, appEnv.IngestRoutes)
	if err != nil {
		return tenancy{}, err
	}
	tenants := tenancy{routes: routes}.orDefaults(db, appEnv, logger)
	if registre.Issuer != "" {
		go tenants.plans.Run(ctx, registre, logger)
	}
	return tenants, nil
}
