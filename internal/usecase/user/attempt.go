package user

import (
	"context"

	"arktie.org/ent"
	"arktie.org/internal/lib/liberrs"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
)

func (uc *Usecase) Attempt(ctx context.Context, session *oauth.ClientSessionData, identity *identity.Identity) (u *ent.User, err error) {
	return nil, liberrs.Todo
}
