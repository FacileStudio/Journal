package main

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/database"
	docs "github.com/FacileStudio/Journal/apps/api/internal/documentation"
	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/middleware"
	"github.com/FacileStudio/Journal/apps/api/modules/alerts"
	"github.com/FacileStudio/Journal/apps/api/modules/antenne"
	"github.com/FacileStudio/Journal/apps/api/modules/apikeys"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/modules/deletions"
	"github.com/FacileStudio/Journal/apps/api/modules/ingest"
	"github.com/FacileStudio/Journal/apps/api/modules/logs"
	"github.com/FacileStudio/Journal/apps/api/modules/queries"
	"github.com/FacileStudio/Journal/apps/api/modules/sourcemaps"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"

	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func main() {
	if healthcheck.Handle(os.Args) {
		return
	}

	os.Exit(run())
}

// run wires the process and serves until a signal arrives. oidc.New is the
// boot path: it performs discovery, so an unreachable or half-configured issuer
// fails here rather than on somebody's first login attempt, while a kit with no
// OIDC_ISSUER is still valid — it serves /auth/config and /auth/logout and
// authenticates every session this app issues. One manager owns one table, and
// both kits issue through it, so a password login and a federated one are the
// same row, the same cookie and the same logout.
func run() int {
	appEnv, err := env.Load()
	appLogger := logger.New(logger.Config{})
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		return 1
	}
	appLogger = logger.New(logger.Config{Level: appEnv.LogLevel})

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		return 1
	}

	if err := schemas.Migrate(db); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return 1
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	identityStore, err := newPorteStore(sqlDB)
	if err != nil {
		appLogger.Error("failed to configure porte token sealing", slog.Any("error", err))
		return 1
	}
	resealInBackground(shutdownSignal, identityStore, appLogger)
	users := auth.NewUserStore(sqlDB)
	if _, err := users.BackfillPictures(shutdownSignal, appEnv.Porte.Issuer); err != nil {
		appLogger.Warn("avatar backfill failed", slog.Any("error", err))
	}

	sessions, err := session.New(appEnv.Porte, session.Deps{
		Sessions: identityStore.Sessions(),
		Logger:   appLogger,
	})
	if err != nil {
		appLogger.Error("failed to configure sessions", slog.Any("error", err))
		return 1
	}

	oidcKit, err := oidc.New(shutdownSignal, appEnv.Porte, oidc.Deps{
		Users:      users,
		Identities: identityStore.Identities(),
		Sessions:   sessions,
		Codes:      identityStore.LoginCodes(),
		Logger:     appLogger,
	})
	if err != nil {
		appLogger.Error("failed to configure authentication", slog.Any("error", err))
		return 1
	}

	if oidcKit.Enabled() {
		appLogger.Info("single sign-on enabled", slog.String("issuer", appEnv.Porte.Issuer))
	}

	if appEnv.RetentionDays > 0 {
		go runRetention(shutdownSignal, db, appEnv.RetentionDays, appLogger)
	} else {
		appLogger.Warn("RETENTION_DAYS is 0: log entries are kept forever, which needs a written justification under GDPR storage-limitation rules; set it to a positive number of days unless you have one")
	}
	go sweepSessions(shutdownSignal, sessions, appLogger)

	var sso ssoRoutes
	if oidcKit.Enabled() {
		registre := directory.Client{
			Issuer:       appEnv.Porte.Issuer,
			ClientID:     appEnv.Porte.ClientID,
			ClientSecret: appEnv.Porte.ClientSecret,
		}
		sso.erasures = deletions.NewService(db, registre, users.EraseAccount, identityStore.EraseIdentity, appLogger)
		sso.invitations = &kit.InvitationsConfig{Directory: registre, Identities: identityStore.Identities(), Logger: appLogger}
		go sso.erasures.Run(shutdownSignal)
	}

	antenneService := antenne.NewService(db, sqlDB, appLogger)
	antenneService.AutoConnect(shutdownSignal)
	defer antenneService.Shutdown()
	go antenneService.Drain(shutdownSignal)

	go alerts.RunEvaluator(shutdownSignal, db, appLogger, appEnv.WebhookAllowedHosts, antenneService)

	router := buildRouter(db, oidcKit, sessions, users, appEnv, appLogger, antenneService, sso, health.DB(sqlDB))

	addr := ":" + strconv.Itoa(appEnv.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	appLogger.Info("server starting", slog.String("addr", addr))
	select {
	case err := <-serverErrCh:
		if !stderrors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server stopped", slog.Any("error", err))
			return 1
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			return 1
		}
		appLogger.Info("server stopped")
	}

	return 0
}

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
func buildRouter(db *gorm.DB, oidcKit *oidc.Kit, sessions *session.Manager, users *kit.Store, appEnv env.Config, appLogger *slog.Logger, antenneService *antenne.Service, sso ssoRoutes, checks ...health.Check) chi.Router {
	ingestService := ingest.NewService(db)
	logsService := logs.NewService(db)
	authService := auth.NewService(db)
	apiKeysService := apikeys.NewService(db)
	queriesService := queries.NewService(db)
	sourceMapsService := sourcemaps.NewService(db)
	alertsService := alerts.NewService(db)

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
	router.Get("/version", versionHandler)
	apiref.Mount(router, referenceConfig())

	requireAuth := kit.RequireAuth(sessions, users)

	router.Route("/api", func(api chi.Router) {
		_ = kit.Mount(api.With(sessionLimiter), kit.Config{Sessions: sessions, OIDC: oidcKit})
		auth.RegisterRoutes(api, authService, sessionLimiter, requireAuth,
			func(w http.ResponseWriter, r *http.Request) { sessions.ClearCookie(w, r, porte.SessionCookieName) })
		sso.register(api.With(sessionLimiter, requireAuth), api, appLogger)
		ingest.RegisterRoutes(api, ingestService, ingestLimiter, middleware.RequireIngestAuth(appEnv.IngestToken, apiKeysService))
		ingest.RegisterBrowserRoutes(api, ingestService, middleware.RequireBrowserIngestAuth(apiKeysService), browserKeyCeiling, browserIngestLimiter)
		sourcemaps.RegisterUploadRoutes(api, sourceMapsService, ingestLimiter, middleware.RequireIngestAuth(appEnv.IngestToken, apiKeysService))

		api.Group(func(protected chi.Router) {
			protected.Use(sessionLimiter)
			protected.Use(requireAuth)
			logs.RegisterRoutes(protected, logsService, sourceMapsService)
			queries.RegisterRoutes(protected, queriesService)
			protected.Group(func(admin chi.Router) {
				admin.Use(middleware.RequireAdmin)
				apikeys.RegisterRoutes(admin, apiKeysService)
				alerts.RegisterRoutes(admin, alertsService)
				sourcemaps.RegisterAdminRoutes(admin, sourceMapsService)
				antenne.RegisterRoutes(admin, antenneService, appLogger)
			})
		})
	})

	if spa.Available(clientDir) {
		router.Handle("/*", spa.Handler(spa.Config{Dir: clientDir}))
		appLogger.Info("serving client", slog.String("dir", clientDir))
	}

	return router
}

// ssoRoutes are the routes that exist only with an OIDC issuer: Registre's
// signed deletion push, and the user menu's pending space invitations.
type ssoRoutes struct {
	erasures    *deletions.Service
	invitations *kit.InvitationsConfig
}

// register mounts the deletion push on the bare API router, since Registre
// signs it and holds no session, and the invitations behind the session.
func (sso ssoRoutes) register(signedIn, api chi.Router, appLogger *slog.Logger) {
	if sso.erasures != nil {
		deletions.RegisterRoutes(api, sso.erasures)
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

// securityConfig extends tronc's default security headers with the inline
// scripts adapter-static writes into the built client, which script-src 'self'
// would otherwise block. A client directory that cannot be hashed is logged and
// leaves the default script policy, which only reports.
func securityConfig(clientDir string, appLogger *slog.Logger) troncmiddleware.SecurityConfig {
	if !spa.Available(clientDir) {
		return troncmiddleware.SecurityConfig{}
	}
	hashes, err := spa.InlineScriptHashes(clientDir)
	if err != nil {
		appLogger.Warn("failed to hash the client's inline scripts", slog.Any("error", err))
		return troncmiddleware.SecurityConfig{}
	}
	return troncmiddleware.SecurityConfig{CSPSources: map[string][]string{"script-src": hashes}}
}

func referenceConfig() apiref.Config {
	return apiref.Config{
		Title:       "Journal API",
		Description: "Centralized logging for the Facile Suite: apps ship structured entries to /ingest, the dashboard searches them.",
		Servers:     []string{"/api"},
		Registry:    docs.Registry,
	}
}

func sweepSessions(ctx context.Context, sessions *session.Manager, logger *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		deleted, err := sessions.Sweep(ctx)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("session sweep failed", slog.Any("error", err))
			}
		} else if deleted > 0 {
			logger.Info("session sweep deleted expired sessions", slog.Int64("deleted", deleted))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func runRetention(ctx context.Context, db *gorm.DB, days int, logger *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		result := db.WithContext(ctx).Exec("DELETE FROM log_entries WHERE created_at < now() - (? * interval '1 day')", days)
		if result.Error != nil {
			if ctx.Err() == nil {
				logger.Error("retention delete failed", slog.Any("error", result.Error))
			}
		} else if result.RowsAffected > 0 {
			logger.Info("retention deleted old log entries",
				slog.Int64("deleted", result.RowsAffected),
				slog.Int("retention_days", days),
			)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
