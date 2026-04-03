package user

import (
	"context"
	"database/sql"
	"testing"
	"testing/synctest"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"

	"arktie.org/ent"
	"arktie.org/ent/enttest"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	_ "modernc.org/sqlite"

	"arktie.org/internal/data/client"
)

func setupUsecase(t *testing.T) *Usecase {
	t.Helper()

	db, err := sql.Open("sqlite", "file:ent?mode=memory&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}

	entClient := enttest.NewClient(t,
		enttest.WithOptions(ent.Driver(entsql.OpenDB("sqlite3", db))),
		enttest.WithMigrateOptions(schema.WithForeignKeys(true)),
	)
	t.Cleanup(func() { entClient.Close() })

	return NewUsecase(&client.Database{Ent: entClient})
}

func TestAttempt_CreateNewUser(t *testing.T) {
	uc := setupUsecase(t)
	ctx := context.Background()

	did := syntax.DID("did:plc:test123")
	session := &oauth.ClientSessionData{
		AccountDID: did,
	}

	u, err := uc.Attempt(ctx, session, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if u.AccountDid != did.String() {
		t.Errorf("expected account_did %s, got %s", did, u.AccountDid)
	}

	if len(u.EncryptionKey) != 32 {
		t.Errorf("expected 32-byte encryption key, got %d bytes", len(u.EncryptionKey))
	}
}

func TestAttempt_ReturnExistingUser(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		uc := setupUsecase(t)
		ctx := context.Background()

		did := syntax.DID("did:plc:test456")
		session := &oauth.ClientSessionData{
			AccountDID: did,
		}

		first, err := uc.Attempt(ctx, session, nil)
		if err != nil {
			t.Fatalf("first attempt: expected no error, got %v", err)
		}

		time.Sleep(time.Second)

		second, err := uc.Attempt(ctx, session, nil)
		if err != nil {
			t.Fatalf("second attempt: expected no error, got %v", err)
		}

		if first.ID != second.ID {
			t.Errorf("expected same user ID, got %s and %s", first.ID, second.ID)
		}

		if !second.LastActiveAt.After(first.LastActiveAt) {
			t.Errorf("expected last_active_at to be updated, first=%v second=%v", first.LastActiveAt, second.LastActiveAt)
		}
	})
}

func TestAttempt_DifferentDIDsCreateDifferentUsers(t *testing.T) {
	uc := setupUsecase(t)
	ctx := context.Background()

	u1, err := uc.Attempt(ctx, &oauth.ClientSessionData{AccountDID: syntax.DID("did:plc:aaa")}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	u2, err := uc.Attempt(ctx, &oauth.ClientSessionData{AccountDID: syntax.DID("did:plc:bbb")}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if u1.ID == u2.ID {
		t.Error("expected different user IDs for different DIDs")
	}
}
