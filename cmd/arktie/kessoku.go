package main

//go:generate go tool kessoku $GOFILE

import (
	"net/http"

	"github.com/mazrean/kessoku"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"arktie.org/internal/data"
	"arktie.org/internal/server"
	"arktie.org/internal/service/oauth"
)

// newServer is the kessoku-generated DI initializer.
// Inputs:  *data.Config	(created in main with special lifecycle handling)
// Output:  *server.Server	(fully wired HTTP server)
//
//nolint:unused
var _ = kessoku.Inject[*http.Server](
	"newServer",
	kessoku.Provide(data.NewClient),
	kessoku.Provide(oauth.NewService),

	kessoku.Bind[server.OAuthHandler](
		kessoku.Provide(func(s *oauth.Service) server.OAuthHandler { return s }),
	),
	kessoku.Provide(server.NewHandler),

	kessoku.Provide(func(cfg *data.Config, handler http.Handler) *http.Server {
		return &http.Server{
			Addr:    cfg.Server.Addr,
			Handler: h2c.NewHandler(handler, &http2.Server{}),
		}
	}),
)
