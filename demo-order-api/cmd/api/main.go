package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/example/demo-incident-response/demo-order-api/internal/handler"
	"github.com/example/demo-incident-response/demo-order-api/internal/middleware"
	"github.com/example/demo-incident-response/demo-order-api/internal/observability"
	"github.com/example/demo-incident-response/demo-order-api/internal/store"
	"github.com/example/demo-incident-response/demo-order-api/observer"
	"github.com/example/demo-incident-response/demo-order-api/web"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Structured JSON logging.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// X-Ray setup.
	if err := observability.ConfigureXRay(); err != nil {
		slog.Warn("failed to configure X-Ray, tracing disabled", "error", err)
	}

	// AWS configuration.
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	// Instrument AWS SDK calls for X-Ray tracing.
	observability.InstrumentAWSConfig(&cfg)

	// Dependencies.
	ddbClient := dynamodb.NewFromConfig(cfg)
	orderStore := store.New(ddbClient)
	orders := handler.NewOrders(orderStore)

	eventsTableName := os.Getenv("EVENTS_TABLE_NAME")
	if eventsTableName == "" {
		eventsTableName = "demo-agent-events"
	}
	eventStore := store.NewEventStore(ddbClient, eventsTableName)
	events := handler.NewEvents(eventStore)

	// Embedded frontend assets.
	staticFS, err := fs.Sub(web.Assets, "static")
	if err != nil {
		slog.Error("failed to load embedded assets", "error", err)
		os.Exit(1)
	}

	mountEventRoutes := func(r chi.Router) {
		r.Get("/agent-events", events.List)
		r.Get("/agent-events/latest", events.Latest)
		r.Get("/agent-events/incidents", events.Incidents)
	}

	baseMiddleware := func(r chi.Router) {
		r.Use(chimw.RequestID)
		r.Use(chimw.RealIP)
		r.Use(middleware.Recovery)
		r.Use(middleware.Logging)
	}

	// Orders router.
	r := chi.NewRouter()
	baseMiddleware(r)
	r.Use(middleware.XRay("demo-order-api"))

	r.Get("/health", handler.Health)
	r.Post("/orders", orders.Create)
	r.Get("/orders", orders.List)
	r.Get("/orders/{id}", orders.Get)
	r.Post("/orders/{id}/refund", orders.Refund)

	r.Route("/api", func(api chi.Router) {
		api.Use(middleware.CORS)
		mountEventRoutes(api)
	})

	// Serve embedded frontend.
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFS, "index.html")
	})

	// Observer router — serves on observer.* subdomain.
	obsRouter := chi.NewRouter()
	baseMiddleware(obsRouter)

	obsRouter.Get("/health", handler.Health)
	obsRouter.Get("/config", handler.Config)
	obsRouter.Route("/api", func(api chi.Router) {
		mountEventRoutes(api)
	})

	observerFS, err := fs.Sub(observer.Assets, "static")
	if err != nil {
		slog.Error("failed to load observer assets", "error", err)
		os.Exit(1)
	}
	obsRouter.Handle("/*", http.FileServer(http.FS(observerFS)))

	// Server.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      hostSwitch(r, obsRouter),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}
}

// hostSwitch routes requests to the observer router when the Host header
// starts with "observer.", otherwise falls through to the orders router.
func hostSwitch(orders, obs http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Host, "observer.") {
			obs.ServeHTTP(w, r)
			return
		}
		orders.ServeHTTP(w, r)
	})
}
