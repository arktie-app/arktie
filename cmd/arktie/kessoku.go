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
	"arktie.org/internal/service/post"
	ucpost "arktie.org/internal/usecase/post"
	"arktie.org/internal/usecase/user"
)

// newServer is the kessoku-generated DI initializer.
// Inputs:  *data.Config	(created in main with special lifecycle handling)
// Output:  *server.Server	(fully wired HTTP server)
//
//nolint:unused
var _ = kessoku.Inject[*App](
	"newApp",
	kessoku.Provide(data.NewClient),

	// usecase
	kessoku.Provide(user.NewUsecase),
	kessoku.Bind[oauth.UserAttempter](
		kessoku.Provide(func(uc *user.Usecase) oauth.UserAttempter { return uc }),
	),

	// usecase - post
	kessoku.Provide(ucpost.NewUsecase),
	kessoku.Bind[post.PostResource](
		kessoku.Provide(func(uc *ucpost.Usecase) post.PostResource { return uc }),
	),

	// services
	kessoku.Provide(oauth.NewService),
	kessoku.Bind[server.OAuthHandler](
		kessoku.Provide(func(s *oauth.Service) server.OAuthHandler { return s }),
	),
	kessoku.Provide(post.NewService),
	kessoku.Bind[server.PostSyncher](
		kessoku.Provide(func(s *post.Service) server.PostSyncher { return s }),
	),

	// handlers
	kessoku.Provide(server.NewHandler),

	// listeners
	kessoku.Provide(server.NewListener),

	// app
	kessoku.Provide(func(cfg *data.Config, client *data.Client, handler http.Handler, _ *server.Listener) *App {
		return &App{
			HTTP: &http.Server{
				Addr:    cfg.Server.Addr,
				Handler: h2c.NewHandler(handler, &http2.Server{}),
			},
			Router: client.Router,
		}
	}),
)
