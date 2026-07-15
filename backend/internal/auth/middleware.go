package auth

import (
	"context"
	"net/http"
	"strings"

	"cloudcostdash/internal/models"
)

type ctxKey string

const claimsCtxKey ctxKey = "claims"

// RequireAuth validates the bearer JWT and attaches its claims to the request context.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := ParseToken(secret, tokenStr)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole rejects the request unless the caller has one of the given roles.
// Must run after RequireAuth.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok || !allowed[claims.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsCtxKey).(*Claims)
	return claims, ok
}
