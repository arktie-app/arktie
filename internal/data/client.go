package data

import (
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	RDB   *redis.Client
	OAuth *oauth.ClientApp
}

func NewClient(cfg *Config) (client *Client, err error) {
	client = &Client{}

	if err = client.initRedisClient(cfg); err != nil {
		return
	}

	if err = client.initOAuthClientApp(cfg); err != nil {
		return
	}

	return
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
		// "repo:app.arktie?action=create"
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

	c.OAuth = oauth.NewClientApp(&clientConfig, oauth.NewMemStore())
	return nil
}
