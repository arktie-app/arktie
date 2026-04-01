package libjwt

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"arktie.org/ent"
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

func (c *Claim) WithUser(user *ent.User) *Claim {
	c.Subject = user.ID.String()
	return c
}

func (c *Claim) WithATSession(session *oauth.ClientSessionData) *Claim {
	c.ATProto.Session = newATProtoSession(session)
	return c
}

func (c *Claim) WithATIdentity(identity *identity.Identity) *Claim {
	c.ATProto.Identity = newATProtoIdentity(identity)
	return c
}

type ATProto struct {
	Session  ATProtoSession  `json:"session"`
	Identity ATProtoIdentity `json:"identity"`
}

type ATProtoSession struct {
	// Account DID for this session. Assuming only one active session per account, this can be used as "primary key" for storing and retrieving this information.
	AccountDID syntax.DID `json:"account_did"`

	// Identifier to distinguish this particular session for the account. Server backends generally support multiple sessions for the same account. This package will re-use the random 'state' token from the auth flow as the session ID.
	SessionID string `json:"session_id"`
}

func newATProtoSession(s *oauth.ClientSessionData) ATProtoSession {
	return ATProtoSession{
		AccountDID: s.AccountDID,
		SessionID:  s.SessionID,
	}
}

type ATProtoIdentity struct {
	// Handle/DID mapping must be bi-directionally verified. If that fails, the Handle should be the special 'handle.invalid' value
	Handle syntax.Handle `json:"handle"`
}

func newATProtoIdentity(i *identity.Identity) ATProtoIdentity {
	return ATProtoIdentity{
		Handle: i.Handle,
	}
}
