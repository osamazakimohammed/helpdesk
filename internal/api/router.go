package api

import (
	"net/http"
	"time"

	"helpdesk/internal/auth"
	"helpdesk/internal/config"
	"helpdesk/internal/db"
	"helpdesk/internal/middleware"
	"helpdesk/internal/storage"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterConfig struct {
	Config     *config.Config
	Queries    db.Querier
	JWTService *auth.JWTService
	Storage    storage.Storage
}

func SetupRouter(rc RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))

	// CORS Config
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-DB-Query-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Query Budget Middleware (<= 5 queries per request)
	r.Use(middleware.QueryBudgetMiddleware(5, false))

	// Instantiate handlers
	authHandler := NewAuthHandler(rc.Queries, rc.JWTService)
	appTicketHandler := NewAppTicketHandler(rc.Queries)
	appEventHandler := NewAppEventHandler(rc.Queries)
	portalHandler := NewPortalHandler(rc.Queries)
	adminHandler := NewAdminHandler(rc.Queries)
	kbHandler := NewKBHandler(rc.Queries)
	submitHandler := NewSubmitHandler(rc.Queries, rc.Storage)

	// API Documentation & OpenAPI 3.1
	r.Get("/api/docs", HandleSwaggerUI)
	r.Get("/openapi.json", HandleOpenAPISpec)

	// Healthchecks
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/openapi.json", HandleOpenAPISpec)

		// 1. Auth routes
		api.Route("/auth", func(authRouter chi.Router) {
			authRouter.Post("/login", authHandler.Login)
			authRouter.Post("/register", authHandler.Register)
			authRouter.Post("/logout", authHandler.Logout)
			authRouter.With(middleware.Authenticate(rc.JWTService)).Get("/me", authHandler.Me)
		})

		// 2. /app - Agent Workspace
		api.Route("/app", func(appRouter chi.Router) {
			appRouter.Use(middleware.Authenticate(rc.JWTService))
			appRouter.Use(middleware.RequireRole("agent", "manager", "admin"))

			appRouter.Get("/tickets", appTicketHandler.ListTickets)
			appRouter.Post("/tickets", appTicketHandler.CreateTicket)
			appRouter.Get("/tickets/{id}", appTicketHandler.GetTicketDetail)
			appRouter.Patch("/tickets/{id}", appTicketHandler.UpdateTicket)

			appRouter.Get("/tickets/{id}/events", appEventHandler.ListEvents)
			appRouter.Post("/tickets/{id}/events", appEventHandler.CreateEvent)

			appRouter.Get("/agents", adminHandler.ListAgents)
			appRouter.Get("/teams", adminHandler.ListTeams)
		})

		// 3. /portal - Customer Portal
		api.Route("/portal", func(portalRouter chi.Router) {
			portalRouter.Use(middleware.Authenticate(rc.JWTService))

			portalRouter.Get("/tickets", portalHandler.ListCustomerTickets)
			portalRouter.Get("/tickets/{id}", portalHandler.GetCustomerTicketDetail)
			portalRouter.Post("/tickets/{id}/reply", portalHandler.ReplyCustomerTicket)
			portalRouter.Post("/tickets/{id}/feedback", portalHandler.SubmitFeedback)
		})

		// 4. /admin - Config Console (Role Gated: Admin, Manager)
		api.Route("/admin", func(adminRouter chi.Router) {
			adminRouter.Use(middleware.Authenticate(rc.JWTService))
			adminRouter.Use(middleware.RequireRole("admin", "manager"))

			adminRouter.Get("/sla-policies", adminHandler.ListSLAPolicies)
			adminRouter.Get("/fields", adminHandler.ListFieldDefinitions)
			adminRouter.Get("/mail-accounts", adminHandler.ListMailAccounts)
			adminRouter.Get("/automation-rules", adminHandler.ListAutomationRules)
			adminRouter.Get("/assignment-rules", adminHandler.ListAssignmentRules)
			adminRouter.Get("/audit-logs", adminHandler.ListAuditLogs)
			adminRouter.Get("/agents", adminHandler.ListAgents)
			adminRouter.Get("/teams", adminHandler.ListTeams)
		})

		// 5. /kb - Knowledge Base (Public, anonymous or authenticated)
		api.Route("/kb", func(kbRouter chi.Router) {
			kbRouter.Get("/spaces", kbHandler.ListSpaces)
			kbRouter.Get("/spaces/{space_id}/categories", kbHandler.ListCategoriesBySpace)
			kbRouter.Get("/categories/{category_id}/articles", kbHandler.ListArticlesByCategory)
			kbRouter.Get("/articles/{slug}", kbHandler.GetArticleBySlug)
			kbRouter.Get("/search", kbHandler.SearchArticles)
			kbRouter.Post("/articles/{id}/feedback", kbHandler.ArticleFeedback)
		})

		// 6. /submit - Anonymous Intake Form
		api.Route("/submit", func(submitRouter chi.Router) {
			submitRouter.Get("/fields", submitHandler.GetIntakeFields)
			submitRouter.Post("/ticket", submitHandler.SubmitTicket)
		})
	})

	return r
}
