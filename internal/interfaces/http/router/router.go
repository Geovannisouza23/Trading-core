// Package router assembles the chi router: middleware chain, public
// endpoints, and the authenticated/RBAC-gated /v1 API.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"trading-core/internal/interfaces/http/handler"
	"trading-core/internal/interfaces/http/middleware"
)

// Config carries every dependency the router needs to wire routes. It is
// built once in internal/app.
type Config struct {
	Logger              *slog.Logger
	JWTSigningSecret    string
	CORSAllowedOrigins  []string
	MaxRequestBodyBytes int64
	RequestTimeout      time.Duration
	RateLimitRPS        float64
	RateLimitBurst      int

	HealthHandler    http.HandlerFunc
	ReadyHandler     http.HandlerFunc
	MetricsHandler   http.Handler
	WebSocketHandler http.Handler

	QueryHandler  *handler.QueryHandler
	SystemHandler *handler.SystemHandler

	BacktestHandler         *handler.BacktestHandler
	OptimizationHandler     *handler.OptimizationHandler
	DatasetHandler          *handler.DatasetHandler
	StrategyHandler         *handler.StrategyHandler
	QuantDiagnosticsHandler *handler.QuantDiagnosticsHandler
}

func New(cfg Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(cfg.Logger))
	r.Use(middleware.Logging(cfg.Logger))
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.BodySizeLimit(cfg.MaxRequestBodyBytes))
	r.Use(middleware.Timeout(cfg.RequestTimeout))
	r.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst))

	r.Get("/health", cfg.HealthHandler)
	r.Get("/ready", cfg.ReadyHandler)
	if cfg.MetricsHandler != nil {
		r.Handle("/metrics", cfg.MetricsHandler)
	}
	if cfg.WebSocketHandler != nil {
		r.Handle("/ws", cfg.WebSocketHandler)
	}

	r.Route("/v1", func(v1 chi.Router) {
		v1.Use(middleware.Auth(cfg.JWTSigningSecret))

		v1.Get("/dashboard", cfg.QueryHandler.Dashboard)
		v1.Get("/account", cfg.QueryHandler.Account)
		v1.Get("/positions", cfg.QueryHandler.Positions)
		v1.Get("/orders", cfg.QueryHandler.Orders)
		v1.Get("/orders/{id}", cfg.QueryHandler.OrderByID)
		v1.Get("/signals", cfg.QueryHandler.Signals)
		v1.Get("/risk-decisions", cfg.QueryHandler.RiskDecisions)
		v1.Get("/events", cfg.QueryHandler.Events)
		v1.Get("/incidents", cfg.QueryHandler.Incidents)

		v1.Group(func(ctrl chi.Router) {
			ctrl.Use(middleware.RequireRole("operator", "admin"))
			ctrl.Post("/system/pause", cfg.SystemHandler.Pause)
			ctrl.Post("/system/resume", cfg.SystemHandler.Resume)
			ctrl.Post("/system/close-only", cfg.SystemHandler.CloseOnly)
			ctrl.Post("/system/kill-switch", cfg.SystemHandler.KillSwitch)
			ctrl.Post("/paper/reset", cfg.SystemHandler.PaperReset)

			ctrl.Post("/backtest/request", cfg.BacktestHandler.Request)
			ctrl.Get("/backtest/{id}", cfg.BacktestHandler.Result)
			ctrl.Get("/backtest/{id}/stream", cfg.BacktestHandler.Stream)

			ctrl.Post("/optimization/request", cfg.OptimizationHandler.Request)
			ctrl.Get("/optimization/{id}", cfg.OptimizationHandler.Result)

			ctrl.Post("/dataset/export", cfg.DatasetHandler.Export)

			ctrl.Post("/strategy/validate", cfg.StrategyHandler.Validate)

			ctrl.Post("/quant/features", cfg.QuantDiagnosticsHandler.CalculateFeatures)
			ctrl.Post("/quant/model/evaluate", cfg.QuantDiagnosticsHandler.EvaluateModel)
			ctrl.Post("/quant/model/reload", cfg.QuantDiagnosticsHandler.ReloadModel)
			ctrl.Get("/quant/feature-schema", cfg.QuantDiagnosticsHandler.FeatureSchema)
			ctrl.Get("/quant/model-metadata", cfg.QuantDiagnosticsHandler.ModelMetadata)
			ctrl.Post("/decisions/{decision_id}/outcome", cfg.QuantDiagnosticsHandler.RegisterDecisionOutcome)

			// Async (NATS fan-out) counterparts to the two sync RPCs above.
			ctrl.Post("/decisions/{decision_id}/outcome/publish", cfg.QuantDiagnosticsHandler.PublishDecisionOutcome)
			ctrl.Post("/quant/model/approved/publish", cfg.QuantDiagnosticsHandler.PublishModelApproved)
		})
	})

	return r
}
