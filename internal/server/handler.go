package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type OAuthHandler interface {
	ClientConfig(http.ResponseWriter, *http.Request)
	Start(http.ResponseWriter, *http.Request)
	Callback(http.ResponseWriter, *http.Request)
}

func NewHandler(oauth OAuthHandler) http.Handler {
	mux := chi.NewMux()

	mux.Group(func(r chi.Router) {
		r.HandleFunc("GET /oauth/client-metadata.json", oauth.ClientConfig)
		r.HandleFunc("GET /oauth/start", oauth.Start)
		r.HandleFunc("GET /oauth/callback", oauth.Callback)
	})

	return mux
}
