package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/libhttp"
	postv1 "arktie.org/internal/pb/post/v1"
	"arktie.org/internal/server/middleware"
	"arktie.org/internal/service/oauth"
	"arktie.org/internal/service/post"
)

func NewHandler(
	cfg *data.Config,
	db *client.Database,

	oauth OAuthHandler,
	postSvc *post.Service,
) http.Handler {
	mux := chi.NewMux()

	mux.Group(func(r chi.Router) {
		r.HandleFunc("GET /oauth/client-metadata.json", oauth.ClientConfig)
		r.HandleFunc("GET /oauth/start", oauth.Start)
		r.HandleFunc("GET /oauth/callback", oauth.Callback)

		r.With(middleware.RequireUser(cfg, db)).
			HandleFunc("POST /api/logout", oauth.Logout)
	})

	mux.With(middleware.RequireUser(cfg, db)).
		HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
			libhttp.WriteJSON(w, http.StatusOK, middleware.UserFromContext(r.Context()))
		})

	gwMux := runtime.NewServeMux()
	postv1.RegisterPostServiceHandlerServer(context.Background(), gwMux, postSvc)

	mux.With(middleware.User(cfg, db)).
		Mount("/", gwMux)

	return mux
}

type OAuthHandler interface {
	ClientConfig(http.ResponseWriter, *http.Request)
	Start(http.ResponseWriter, *http.Request)
	Callback(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
}

var _ OAuthHandler = &oauth.Service{}
