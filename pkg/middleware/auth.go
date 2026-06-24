package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-core-service/pkg/httputil"
	"github.com/vsayfb/gig-platform-core-service/pkg/jwt"
)

func Auth(jwtManager *jwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)

			if len(parts) != 2 || parts[0] != "Bearer" {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			claims, err := jwtManager.Verify(parts[1])

			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			userID, err := uuid.Parse(claims.Subject)

			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid token subject")
				return
			}

			ctx := context.WithValue(r.Context(), "userID", userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
