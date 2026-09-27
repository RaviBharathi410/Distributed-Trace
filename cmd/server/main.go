package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/api"
	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/config"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
	pgRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Structured Logger
	observability.InitLogger(cfg.Environment)
	defer func() {
		_ = observability.Log.Sync()
	}()

	observability.Log.Info("Starting DistributedTrace Server",
		zap.Int("port", cfg.ServerPort),
		zap.String("environment", cfg.Environment),
		zap.String("log_level", cfg.LogLevel),
	)

	// 3. Initialize OpenTelemetry Tracer (non-blocking if collector not yet reachable)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "distributedtrace-api", "")
	if err != nil {
		observability.Log.Warn("Failed to initialize OpenTelemetry tracer (continuing without remote trace exporter)",
			zap.Error(err),
		)
	} else if shutdownTracer != nil {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := shutdownTracer(shutdownCtx); err != nil {
				observability.Log.Error("Error shutting down tracer", zap.Error(err))
			}
		}()
	}

	// 4. Initialize Database Connections (PostgreSQL & ClickHouse)
	pgPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		observability.Log.Warn("Failed to create PostgreSQL pool", zap.Error(err))
	} else {
		pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
		if err := pgPool.Ping(pingCtx); err != nil {
			observability.Log.Warn("PostgreSQL ping failed (database not yet reachable)", zap.Error(err))
		} else {
			observability.Log.Info("Connected to PostgreSQL")
		}
		pingCancel()
		defer pgPool.Close()
	}

	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.ClickHouseURL},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDB,
			Username: "default",
			Password: "",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		observability.Log.Warn("Failed to create ClickHouse connection", zap.Error(err))
	} else {
		pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
		if err := chConn.Ping(pingCtx); err != nil {
			observability.Log.Warn("ClickHouse ping failed (database not yet reachable)", zap.Error(err))
		} else {
			observability.Log.Info("Connected to ClickHouse")
		}
		pingCancel()
		defer chConn.Close()
	}

	// 5. Initialize Token Service
	tokenService, err := auth.NewTokenServiceFromPaths(
		cfg.JWTPrivateKeyPath,
		cfg.JWTPublicKeyPath,
		"distributedtrace-api",
		cfg.Environment,
	)
	if err != nil {
		observability.Log.Fatal("Failed to initialize TokenService", zap.Error(err))
	}

	// 6. Initialize Repositories
	userRepo := pgRepo.NewUserRepository(pgPool)
	orgRepo := pgRepo.NewOrganizationRepository(pgPool)
	apiKeyRepo := pgRepo.NewAPIKeyRepository(pgPool)

	traceRepo := chRepo.NewTraceRepository(chConn)
	serviceRepo := chRepo.NewServiceRepository(chConn)
	anomalyRepo := chRepo.NewAnomalyRepository(chConn)

	// 7. Initialize API Handlers
	authHandler := api.NewAuthHandler(userRepo, orgRepo, apiKeyRepo, tokenService)
	traceHandler := api.NewTraceHandler(traceRepo)
	serviceHandler := api.NewServiceHandler(serviceRepo)
	anomalyHandler := api.NewAnomalyHandler(anomalyRepo)
	spanHandler := api.NewSpanHandler(traceRepo, nil)

	// 8. Setup Chi Router & Middleware
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.ServerWriteTimeout))

	// CORS Middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")
			// Strict allowlist: validate against configured AllowedOrigin and local dev hosts
			if origin != "" && (origin == cfg.AllowedOrigin || (cfg.Environment == "development" && (origin == "http://localhost:5173" || origin == "http://localhost:3000" || origin == "http://127.0.0.1:5173"))) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if cfg.AllowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", cfg.AllowedOrigin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, X-API-Key")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Request Logging & Prometheus Metrics Middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)

			next.ServeHTTP(ww, req)

			duration := time.Since(start)
			statusStr := fmt.Sprintf("%d", ww.Status())

			observability.HttpRequestsTotal.WithLabelValues(req.Method, req.URL.Path, statusStr).Inc()
			observability.HttpRequestDuration.WithLabelValues(req.Method, req.URL.Path).Observe(duration.Seconds())

			observability.Log.Debug("HTTP Request",
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Duration("duration", duration),
				zap.String("req_id", middleware.GetReqID(req.Context())),
			)
		})
	})

	// 9. System Routes
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"service":     "distributedtrace-api",
			"environment": cfg.Environment,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.Handle("/metrics", promhttp.Handler())

	// 10. Mount REST API Routes (/api/v1)
	r.Route("/api/v1", func(r chi.Router) {
		// Auth endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)

			// Protected user session endpoints
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireUserAuth(tokenService))
				r.Get("/me", authHandler.Me)
				r.Get("/api-keys", authHandler.ListAPIKeys)

				// Sensitive API key management requires Admin or Owner
				r.With(auth.RequireRole(domain.RoleAdmin)).Post("/api-keys", authHandler.CreateAPIKey)
				r.With(auth.RequireRole(domain.RoleAdmin)).Delete("/api-keys/{id}", authHandler.RevokeAPIKey)
			})
		})

		// Span Ingestion (Dual-mode: API Key or User JWT)
		r.Route("/spans", func(r chi.Router) {
			r.Use(auth.RequireIngestAuth(apiKeyRepo, tokenService))
			r.Post("/", spanHandler.IngestSpans)
		})

		// Traces endpoints (Protected)
		r.Route("/traces", func(r chi.Router) {
			r.Use(auth.RequireUserAuth(tokenService))
			r.Get("/", traceHandler.SearchTraces)
			r.Get("/{trace_id}", traceHandler.GetTraceByID)
		})

		// Services endpoints (Protected)
		r.Route("/services", func(r chi.Router) {
			r.Use(auth.RequireUserAuth(tokenService))
			r.Get("/", serviceHandler.ListServices)
			r.Get("/graph", serviceHandler.GetServiceGraph)
			r.Get("/{service}/stats", serviceHandler.GetServiceStats)
		})

		// Anomalies endpoints (Protected)
		r.Route("/anomalies", func(r chi.Router) {
			r.Use(auth.RequireUserAuth(tokenService))
			r.Get("/", anomalyHandler.ListAnomalies)
			r.Get("/{id}", anomalyHandler.GetAnomalyByID)

			// Mutating anomaly status requires Member, Admin, or Owner
			r.With(auth.RequireRole(domain.RoleMember, domain.RoleAdmin)).Patch("/{id}/status", anomalyHandler.UpdateAnomalyStatus)
		})
	})

	// 11. Start HTTP Server
	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		observability.Log.Info("Server listening", zap.String("address", serverAddr))
		serverErrors <- srv.ListenAndServe()
	}()

	// 7. Graceful Shutdown on SIGINT / SIGTERM
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			observability.Log.Fatal("Server failed to start", zap.Error(err))
		}
	case sig := <-shutdown:
		observability.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			observability.Log.Error("Graceful shutdown failed, forcing close", zap.Error(err))
			_ = srv.Close()
		}
		observability.Log.Info("Server stopped cleanly")
	}
}
