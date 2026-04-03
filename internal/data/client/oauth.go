package client

import (
	"github.com/bluesky-social/indigo/atproto/auth/oauth"

	"arktie.org/internal/data"
	"arktie.org/pkg/atproto"
)

type OAuth struct {
	ATProto *oauth.ClientApp
}

func NewOAuth(cfg *data.Config, db *Database) (client *OAuth, err error) {
	client = &OAuth{}

	if err = client.initATProtoClientApp(cfg, db); err != nil {
		return
	}

	return
}

func (c *OAuth) initATProtoClientApp(cfg *data.Config, db *Database) error {
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

	c.ATProto = oauth.NewClientApp(&clientConfig, atproto.NewRedisStore(db.RDB))
	return nil
}
