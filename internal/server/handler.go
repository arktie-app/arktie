package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/libhttp"
	"arktie.org/internal/server/middleware"
)

type OAuthHandler interface {
	ClientConfig(http.ResponseWriter, *http.Request)
	Start(http.ResponseWriter, *http.Request)
	Callback(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
}

func NewHandler(cfg *data.Config, oauth OAuthHandler) http.Handler {
	mux := chi.NewMux()

	mux.Group(func(r chi.Router) {
		r.HandleFunc("GET /oauth/client-metadata.json", oauth.ClientConfig)
		r.HandleFunc("GET /oauth/start", oauth.Start)
		r.HandleFunc("GET /oauth/callback", oauth.Callback)

		r.With(middleware.User(cfg)).
			HandleFunc("POST /api/logout", oauth.Logout)
	})

	mux.With(middleware.RequireUser(cfg)).
		HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
			libhttp.WriteJSON(w, http.StatusOK, middleware.UserFromContext(r.Context()))
		})

	return mux
}
