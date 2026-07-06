package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/database"
	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
	"github.com/FacileStudio/Journal/apps/api/internal/logger"
	"github.com/FacileStudio/Journal/apps/api/internal/middleware"
	"github.com/FacileStudio/Journal/apps/api/modules/apikeys"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/modules/ingest"
	"github.com/FacileStudio/Journal/apps/api/modules/logs"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"gorm.io/gorm"
)

func main() {
	appEnv, err := env.Load()
	appLogger := logger.New("info")
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}
	appLogger = logger.New(appEnv.LogLevel)

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		os.Exit(1)
	}

	if err := schemas.Migrate(db); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if appEnv.RetentionDays > 0 {
		go runRetention(shutdownSignal, db, appEnv.RetentionDays, appLogger)
	}

	ingestService := ingest.NewService(db)
	logsService := logs.NewService(db)
	authService := auth.NewService(db)
	apiKeysService := apikeys.NewService(db)

	credentialLimiter := httprate.Limit(20, time.Minute, httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint))
	sessionLimiter := httprate.LimitByIP(300, time.Minute)
	ingestLimiter := httprate.Limit(600, time.Minute, httprate.WithKeyFuncs(middleware.KeyByBearerTokenHash))

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.CORS(appEnv.AllowedOrigins))
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.RequestLogger(appLogger))
	router.Use(chimiddleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessContext, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(readinessContext); err != nil {
			httpjson.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	auth.RegisterRoutes(router, authService, appEnv.AllowRegistration, credentialLimiter, sessionLimiter)
	ingest.RegisterRoutes(router, ingestService, ingestLimiter, middleware.RequireIngestAuth(appEnv.IngestToken, apiKeysService))

	router.Group(func(protected chi.Router) {
		protected.Use(sessionLimiter)
		protected.Use(middleware.RequireAuth(authService))
		logs.RegisterRoutes(protected, logsService)
		protected.Group(func(admin chi.Router) {
			admin.Use(middleware.RequireAdmin)
			apikeys.RegisterRoutes(admin, apiKeysService)
		})
	})

	addr := ":" + appEnv.Port
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
		if !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server stopped", slog.Any("error", err))
			os.Exit(1)
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			os.Exit(1)
		}
		appLogger.Info("server stopped")
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
