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
	"arktie.org/internal/usecase/user"
)

// newServer is the kessoku-generated DI initializer.
// Inputs:  *data.Config	(created in main with special lifecycle handling)
// Output:  *server.Server	(fully wired HTTP server)
//
//nolint:unused
var _ = kessoku.Inject[*http.Server](
	"newServer",
	kessoku.Provide(data.NewClient),

	// usecase
	kessoku.Provide(user.NewUsecase),
	kessoku.Bind[oauth.UserAttempter](
		kessoku.Provide(func(uc *user.Usecase) oauth.UserAttempter { return uc }),
	),

	// services
	kessoku.Provide(oauth.NewService),
	kessoku.Bind[server.OAuthHandler](
		kessoku.Provide(func(s *oauth.Service) server.OAuthHandler { return s }),
	),

	// handlers
	kessoku.Provide(server.NewHandler),

	// server
	kessoku.Provide(func(cfg *data.Config, handler http.Handler) *http.Server {
		return &http.Server{
			Addr:    cfg.Server.Addr,
			Handler: h2c.NewHandler(handler, &http2.Server{}),
		}
	}),
)
