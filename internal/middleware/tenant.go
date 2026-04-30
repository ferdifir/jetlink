package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ferdifir/jetlink/internal/repository"
)

type contextKey string

const TenantKey contextKey = "tenant"

func TenantMiddleware(repo *repository.TenantRepo) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := mux.Vars(r)["slug"]
			if slug == "" {
				http.NotFound(w, r)
				return
			}

			tenant, err := repo.GetBySlug(slug)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if tenant == nil {
				http.NotFound(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), TenantKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
