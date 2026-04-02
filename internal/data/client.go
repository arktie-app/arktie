package data

import (
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/XSAM/otelsql"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"

	"arktie.org/ent"
	"arktie.org/pkg/atproto"
)

type Client struct {
	Ent   *ent.Client
	RDB   *redis.Client
	OAuth *oauth.ClientApp

	Publisher  message.Publisher
	Subscriber message.Subscriber
	Router     *message.Router
}

func NewClient(cfg *Config) (client *Client, err error) {
	client = &Client{}

	if err = client.initSQLClient(cfg); err != nil {
		return
	}

	if err = client.initRedisClient(cfg); err != nil {
		return
	}

	if err = client.initOAuthClientApp(cfg); err != nil {
		return
	}

	if err = client.initWatermill(cfg); err != nil {
		return
	}

	return
}

func (c *Client) initSQLClient(cfg *Config) error {
	db, err := otelsql.Open(
		cfg.Service.Database.SQL.Driver, cfg.Service.Database.SQL.DSN,
		otelsql.WithAttributes(semconv.DBSystemNameKey.String(cfg.Service.Database.SQL.Driver)),
	)
	if err != nil {
		return err
	}

	c.Ent = ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	return db.Ping()
}

func (c *Client) initRedisClient(cfg *Config) (err error) {
	c.RDB = redis.NewClient(&redis.Options{
		Addr:     cfg.Service.Database.Redis.Addr,
		Password: cfg.Service.Database.Redis.Password,
	})

	return redisotel.InstrumentTracing(c.RDB)
}

func (c *Client) initOAuthClientApp(cfg *Config) error {
	scopes := []string{
		"atproto",
		"repo:app.arktie.post",
	}

	var clientConfig oauth.ClientConfig
	if cfg.App.IsLocal() {
		clientConfig = oauth.NewLocalhostConfig(cfg.App.URL.JoinPath("/oauth/callback").String(), scopes)
	} else {
		clientConfig = oauth.NewPublicConfig(
			cfg.App.URL.JoinPath("oauth/client-metadata.json").String(),
			cfg.App.URL.JoinPath("oauth/callback").String(),
			scopes,
		)
	}

	c.OAuth = oauth.NewClientApp(&clientConfig, atproto.NewRedisStore(c.RDB))
	return nil
}

func (c *Client) initWatermill(cfg *Config) error {
	logger := watermill.NewStdLogger(cfg.App.Debug, cfg.App.Debug)

	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	c.Publisher = pubSub
	c.Subscriber = pubSub

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return err
	}

	// SignalsHandler will gracefully shutdown Router when SIGTERM is received.
	// You can also close the router by just calling `r.Close()`.
	router.AddPlugin(plugin.SignalsHandler)

	// Router level middleware are executed for every message sent to the router
	router.AddMiddleware(
		// CorrelationID will copy the correlation id from the incoming message's metadata to the produced messages
		middleware.CorrelationID,

		// The handler function is retried if it returns an error.
		// After MaxRetries, the message is Nacked and it's up to the PubSub to resend it.
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: time.Millisecond * 100,
			Logger:          logger,
		}.Middleware,

		// Recoverer handles panics from handlers.
		// In this case, it passes them as errors to the Retry middleware.
		middleware.Recoverer,
	)

	c.Router = router
	return nil
}
