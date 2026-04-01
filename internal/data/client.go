package data

import "github.com/bluesky-social/indigo/atproto/auth/oauth"

type Client struct {
	OAuth *oauth.ClientApp
}

func NewClient(cfg *Config) (client *Client, err error) {
	client = &Client{}

	client.initOAuthClientApp(cfg)

	return
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
			cfg.App.URL.JoinPath("/oauth/callback").String(),
			scopes,
		)
	}

	c.OAuth = oauth.NewClientApp(&clientConfig, oauth.NewMemStore())
	return nil
}
