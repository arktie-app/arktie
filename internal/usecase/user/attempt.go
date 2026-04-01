package user

import (
	"context"
	"crypto/rand"

	"arktie.org/ent"
	"arktie.org/ent/user"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
)

func (uc *Usecase) Attempt(ctx context.Context, session *oauth.ClientSessionData, identity *identity.Identity) (u *ent.User, err error) {
	u, err = uc.client.Ent.User.Query().
		Where(user.AccountDid(session.AccountDID.String())).
		Only(ctx)
	if err == nil {
		return u, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	key, err := makeEncryptionKey()
	if err != nil {
		return nil, err
	}

	return uc.client.Ent.User.Create().
		SetAccountDid(session.AccountDID.String()).
		SetEncryptionKey(key).
		Save(ctx)
}

func makeEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // For AES-256, it needs 32 bytes key

	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}
