package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const claimsKey contextKey = "auth_claims"

// Claims is the minimal identity/RBAC information extracted from a valid
// JWT: who (Subject) and what they're allowed to do (Roles).
type Claims struct {
	Subject string
	Roles   []string
}

// Auth validates a bearer JWT signed with signingSecret (HS256) and stores
// its claims on the request context. It never inspects a database or
// external identity provider — the secret comes from SecurityConfig.
func Auth(signingSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			tokenString, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || tokenString == "" {
				writeUnauthorized(w)
				return
			}

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				return []byte(signingSecret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !token.Valid {
				writeUnauthorized(w)
				return
			}
			mapClaims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeUnauthorized(w)
				return
			}

			subject, _ := mapClaims["sub"].(string)
			var roles []string
			if rawRoles, ok := mapClaims["roles"].([]any); ok {
				for _, raw := range rawRoles {
					if s, ok := raw.(string); ok {
						roles = append(roles, s)
					}
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, Claims{Subject: subject, Roles: roles})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves the Claims stored by Auth, if any.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}

// RequireRole only allows requests whose JWT claims include at least one of
// the given roles (RBAC).
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeUnauthorized(w)
				return
			}
			for _, role := range claims.Roles {
				if allowed[role] {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"insufficient role"}}`))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"missing or invalid bearer token"}}`))
}
