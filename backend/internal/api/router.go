package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"

	appauth "cloudcostdash/internal/auth"
	"cloudcostdash/internal/crypto"
	"cloudcostdash/internal/models"
	"cloudcostdash/internal/scheduler"
)

type Server struct {
	DB        *gorm.DB
	Box       *crypto.Box
	JWTSecret string
	Scheduler *scheduler.Scheduler
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	r.Post("/api/auth/login", s.handleLogin)

	r.Group(func(r chi.Router) {
		r.Use(appauth.RequireAuth(s.JWTSecret))

		r.Get("/api/me", s.handleMe)

		// hierarchy: read allowed for any authenticated role, viewer results
		// are scope-filtered inside the handlers themselves.
		r.Get("/api/tree", s.handleTree)

		r.Get("/api/costs/timeseries", s.handleCostTimeseries)

		r.Group(func(r chi.Router) {
			r.Use(appauth.RequireRole(models.RoleAdmin))

			r.Post("/api/users", s.handleCreateUser)
			r.Get("/api/users", s.handleListUsers)

			r.Post("/api/organizations", s.handleCreateOrganization)
			r.Delete("/api/organizations/{id}", s.handleDeleteOrganization)

			r.Post("/api/products", s.handleCreateProduct)
			r.Delete("/api/products/{id}", s.handleDeleteProduct)

			r.Post("/api/projects", s.handleCreateProject)
			r.Delete("/api/projects/{id}", s.handleDeleteProject)

			r.Post("/api/environments", s.handleCreateEnvironment)
			r.Delete("/api/environments/{id}", s.handleDeleteEnvironment)

			r.Post("/api/regions", s.handleCreateRegion)
			r.Delete("/api/regions/{id}", s.handleDeleteRegion)

			r.Post("/api/credentials", s.handleCreateCredential)
			r.Get("/api/credentials", s.handleListCredentials)
			r.Delete("/api/credentials/{id}", s.handleDeleteCredential)

			r.Post("/api/cloud-accounts", s.handleCreateCloudAccount)
			r.Patch("/api/cloud-accounts/{id}", s.handleUpdateCloudAccount)
			r.Delete("/api/cloud-accounts/{id}", s.handleDeleteCloudAccount)
			r.Post("/api/cloud-accounts/{id}/sync-now", s.handleSyncNow)
			r.Post("/api/cloud-accounts/{id}/import-csv", s.handleImportCSV)
		})
	})

	return r
}
