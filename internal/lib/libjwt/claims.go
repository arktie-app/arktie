package libjwt

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"arktie.org/internal/data"
)

type Claim struct {
	jwt.RegisteredClaims
	ATProto `json:"atp"`
}

func NewClaim(cfg *data.Config) *Claim {
	now := time.Now()

	c := &Claim{}

	c.Issuer = cfg.App.URL.Hostname()
	c.Audience = jwt.ClaimStrings{
		cfg.App.URL.Hostname(),
	}
	c.ExpiresAt = jwt.NewNumericDate(now.AddDate(0, 0, 14))
	c.NotBefore = jwt.NewNumericDate(now)
	c.IssuedAt = jwt.NewNumericDate(now)
	c.ID = uuid.Must(uuid.NewV7()).String()

	return c
}

func (c *Claim) WithATSession(session *oauth.ClientSessionData) *Claim {
	c.Subject = session.AccountDID.String()
	c.ATProto.Session = session
	return c
}

func (c *Claim) WithATIdentity(identity *identity.Identity) *Claim {
	c.ATProto.Identity = identity
	return c
}

type ATProto struct {
	Session  *oauth.ClientSessionData `json:"session"`
	Identity *identity.Identity       `json:"identity"`
}
