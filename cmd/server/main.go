package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"helpdesk/internal/api"
	"helpdesk/internal/assignment"
	"helpdesk/internal/auth"
	"helpdesk/internal/automation"
	"helpdesk/internal/config"
	"helpdesk/internal/db"
	"helpdesk/internal/dbconn"
	"helpdesk/internal/email"
	"helpdesk/internal/outbox"
	"helpdesk/internal/storage"
	"helpdesk/internal/webhook"
	"helpdesk/web"
	"github.com/go-chi/chi/v5"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "migrate":
		runMigrations(ctx, cfg)
	case "seed":
		runSeed(ctx, cfg)
	case "worker":
		runWorkerOnly(ctx, cfg)
	case "serve":
		fallthrough
	default:
		runServer(ctx, cfg)
	}
}

func runMigrations(ctx context.Context, cfg *config.Config) {
	slog.Info("Running database migrations...")
	pool, err := dbconn.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database for migrations", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	schemaSQL, err := os.ReadFile("migrations/000001_schema.up.sql")
	if err != nil {
		slog.Error("Failed to read schema migration file", "error", err)
		os.Exit(1)
	}

	_, err = pool.Exec(ctx, string(schemaSQL))
	if err != nil {
		slog.Error("Failed to execute schema migration", "error", err)
		os.Exit(1)
	}

	slog.Info("Schema migrations executed successfully!")
}

func runSeed(ctx context.Context, cfg *config.Config) {
	slog.Info("Seeding initial data...")
	pool, err := dbconn.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database for seed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	seedSQL, err := os.ReadFile("migrations/000002_seed.up.sql")
	if err != nil {
		slog.Error("Failed to read seed file", "error", err)
		os.Exit(1)
	}

	_, err = pool.Exec(ctx, string(seedSQL))
	if err != nil {
		slog.Error("Failed to execute seed data", "error", err)
		os.Exit(1)
	}

	slog.Info("Seed data inserted successfully!")
}

func runWorkerOnly(ctx context.Context, cfg *config.Config) {
	slog.Info("Starting standalone outbox worker...")
	pool, err := dbconn.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	outboxWorker := setupOutboxWorker(cfg, queries)
	outboxWorker.Start(ctx)

	waitForShutdown(cancelContext)
}

func runServer(ctx context.Context, cfg *config.Config) {
	slog.Info("Starting Helpdesk Platform server...", "env", cfg.AppEnv, "port", cfg.Port)

	pool, err := dbconn.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Warn("Database connection failed at startup", "error", err)
	}

	var queries db.Querier
	if pool != nil {
		queries = db.New(pool)
	}

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiry)
	storageProvider, _ := storage.InitStorage(cfg)

	if queries != nil {
		outboxWorker := setupOutboxWorker(cfg, queries)
		outboxWorker.Start(ctx)
	}

	r := api.SetupRouter(api.RouterConfig{
		Config:     cfg,
		Queries:    queries,
		JWTService: jwtService,
		Storage:    storageProvider,
	})

	mountStaticFrontend(r)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	slog.Info(fmt.Sprintf("Helpdesk is live at %s", cfg.BaseURL))
	slog.Info(fmt.Sprintf("Surfaces:\n - Agent Workspace: %s/app\n - Customer Portal: %s/portal\n - Admin Console: %s/admin\n - Knowledge Base: %s/kb\n - Intake Form: %s/submit\n - OpenAPI Docs: %s/api/docs", cfg.BaseURL, cfg.BaseURL, cfg.BaseURL, cfg.BaseURL, cfg.BaseURL, cfg.BaseURL))

	waitForShutdown(func() {
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer sCancel()
		_ = srv.Shutdown(shutdownCtx)
		if pool != nil {
			pool.Close()
		}
	})
}

func setupOutboxWorker(cfg *config.Config, queries db.Querier) *outbox.Worker {
	worker := outbox.NewWorker(queries, 1*time.Second)
	emailEngine := email.NewEngine(cfg, queries)
	webhookDispatcher := webhook.NewDispatcher(queries)
	autoEngine := automation.NewEngine(queries)
	_ = assignment.NewEngine(queries)

	worker.RegisterHandler(outbox.EventTicketCreated, func(ctx context.Context, item db.Outbox) error {
		slog.Info("Outbox: Processing Ticket Created", "id", item.AggregateID)
		_ = webhookDispatcher.DispatchEvent(ctx, outbox.EventTicketCreated, item.Payload)
		return nil
	})

	worker.RegisterHandler(outbox.EventTicketUpdated, func(ctx context.Context, item db.Outbox) error {
		slog.Info("Outbox: Processing Ticket Updated", "id", item.AggregateID)
		_ = webhookDispatcher.DispatchEvent(ctx, outbox.EventTicketUpdated, item.Payload)
		return nil
	})

	worker.RegisterHandler(outbox.EventReplyReceived, func(ctx context.Context, item db.Outbox) error {
		slog.Info("Outbox: Processing Reply Received", "id", item.AggregateID)
		_ = webhookDispatcher.DispatchEvent(ctx, outbox.EventReplyReceived, item.Payload)
		return nil
	})

	_ = emailEngine
	_ = autoEngine

	return worker
}

func mountStaticFrontend(r *chi.Mux) {
	distFS, _ := web.GetDistFS()

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if path == "/api/docs" || path == "/api/v1/openapi.json" || path == "/openapi.json" || path == "/healthz" {
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		// 1. Try serving from local disk first if exists (useful for dev and hot reload)
		diskPath := "./web/dist/" + cleanPath
		if fi, err := os.Stat(diskPath); err == nil && !fi.IsDir() {
			http.ServeFile(w, req, diskPath)
			return
		}

		// 2. Try embedded FS
		if distFS != nil {
			if data, err := fs.ReadFile(distFS, cleanPath); err == nil {
				ext := strings.ToLower(cleanPath)
				if strings.HasSuffix(ext, ".css") {
					w.Header().Set("Content-Type", "text/css; charset=utf-8")
				} else if strings.HasSuffix(ext, ".js") {
					w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
				} else if strings.HasSuffix(ext, ".json") {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
				} else if strings.HasSuffix(ext, ".svg") {
					w.Header().Set("Content-Type", "image/svg+xml")
				} else if strings.HasSuffix(ext, ".html") {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
		}

		// 3. Fallback to index.html for SPA client routes (/app, /portal, /admin, /kb, /submit, /login)
		if diskIndex, err := os.ReadFile("./web/dist/index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(diskIndex)
			return
		}

		if distFS != nil {
			if indexData, err := fs.ReadFile(distFS, "index.html"); err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(indexData)
				return
			}
		}

		http.NotFound(w, req)
	})
}

func cancelContext() {}

func waitForShutdown(onShutdown func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	slog.Info("Shutting down gracefully...")
	onShutdown()
	slog.Info("Helpdesk server stopped.")
}
