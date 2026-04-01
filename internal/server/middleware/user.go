package middleware

import (
	"context"
	"net/http"
	"strings"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/libhttp"
	"arktie.org/internal/lib/libjwt"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey struct{}

// User is a chi middleware that extracts a JWT from the Authorization header
// (Bearer token) or the session cookie, validates it, and stores the parsed
// claim in the request context.
//
// It does not block request even if user is not authenticated.
func User(cfg *data.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if claim, err := authenticate(cfg, r); err == nil {
				ctx := context.WithValue(r.Context(), ctxKey{}, claim)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireUser is a chi middleware that extract a JWT from the Authorization header
// (Bearer token) or the session cookie, validates it, and stores the parsed
// claim in the request context.
//
// It bloks with response "401 Unauthorized" if user is not authenticated.
func RequireUser(cfg *data.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claim, err := authenticate(cfg, r)
			if err != nil {
				libhttp.WriteError(w, http.StatusUnauthorized, "invalid or missing token")
				return
			}

			ctx := context.WithValue(r.Context(), ctxKey{}, claim)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext returns the JWT claim stored in the context by the User
// middleware. Returns nil if no valid token was present.
func UserFromContext(ctx context.Context) *libjwt.Claim {
	claim, _ := ctx.Value(ctxKey{}).(*libjwt.Claim)
	return claim
}

func tokenFromHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return after
	}
	return ""
}

func tokenFromCookie(r *http.Request) string {
	c, err := r.Cookie(data.SessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func authenticate(cfg *data.Config, r *http.Request) (*libjwt.Claim, error) {
	tokenString := tokenFromHeader(r)
	if tokenString == "" {
		tokenString = tokenFromCookie(r)
	}

	if tokenString == "" {
		return nil, jwt.ErrTokenMalformed
	}

	claim := &libjwt.Claim{}
	token, err := jwt.ParseWithClaims(tokenString, claim, func(t *jwt.Token) (any, error) {
		return cfg.App.Key, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(cfg.App.URL.Hostname()),
		jwt.WithAudience(cfg.App.URL.Hostname()),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claim, nil
}
