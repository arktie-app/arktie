package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewHandler() http.Handler {
	mux := chi.NewMux()

	return mux
}
