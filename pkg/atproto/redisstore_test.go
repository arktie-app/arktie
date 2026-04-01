package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_Session(t *testing.T) {
	ctx := context.Background()
	db, mock := redismock.NewClientMock()
	store := NewRedisStore(db)

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	did := syntax.DID("did:plc:123")
	sessionID := "sess-123"
	key := sessionKey(did, sessionID)

	sess := oauth.ClientSessionData{
		AccountDID: did,
		SessionID:  sessionID,
	}
	data, _ := json.Marshal(sess)

	t.Run("GetSession_Success", func(t *testing.T) {
		mock.ExpectGet(key).SetVal(string(data))

		got, err := store.GetSession(ctx, did, sessionID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if got.SessionID != sessionID {
			t.Errorf("expected sessionID %s, got %s", sessionID, got.SessionID)
		}
	})

	t.Run("GetSession_NotFound", func(t *testing.T) {
		mock.ExpectGet(key).RedisNil()

		_, err := store.GetSession(ctx, did, sessionID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("GetSession_RedisError", func(t *testing.T) {
		mock.ExpectGet(key).SetErr(fmt.Errorf("redis down"))

		_, err := store.GetSession(ctx, did, sessionID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("GetSession_InvalidJSON", func(t *testing.T) {
		mock.ExpectGet(key).SetVal("invalid-json")

		_, err := store.GetSession(ctx, did, sessionID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("SaveSession_Success", func(t *testing.T) {
		mock.ExpectSet(key, data, SessionTTL).SetVal("OK")

		err := store.SaveSession(ctx, sess)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("SaveSession_RedisError", func(t *testing.T) {
		mock.ExpectSet(key, data, SessionTTL).SetErr(fmt.Errorf("redis down"))

		err := store.SaveSession(ctx, sess)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("DeleteSession_Success", func(t *testing.T) {
		mock.ExpectDel(key).SetVal(1)

		err := store.DeleteSession(ctx, did, sessionID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestRedisStore_AuthRequestInfo(t *testing.T) {
	ctx := context.Background()
	db, mock := redismock.NewClientMock()
	store := NewRedisStore(db)

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	state := "state-123"
	key := authRequestKey(state)

	info := oauth.AuthRequestData{
		State: state,
	}
	data, _ := json.Marshal(info)

	t.Run("GetAuthRequestInfo_Success", func(t *testing.T) {
		mock.ExpectGet(key).SetVal(string(data))

		got, err := store.GetAuthRequestInfo(ctx, state)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if got.State != state {
			t.Errorf("expected state %s, got %s", state, got.State)
		}
	})

	t.Run("GetAuthRequestInfo_NotFound", func(t *testing.T) {
		mock.ExpectGet(key).RedisNil()

		_, err := store.GetAuthRequestInfo(ctx, state)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("GetAuthRequestInfo_InvalidJSON", func(t *testing.T) {
		mock.ExpectGet(key).SetVal("invalid-json")

		_, err := store.GetAuthRequestInfo(ctx, state)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("SaveAuthRequestInfo_Success", func(t *testing.T) {
		mock.ExpectSetArgs(key, data, redis.SetArgs{
			Mode: "NX",
			TTL:  AuthRequestTTL,
		}).SetVal("OK")

		err := store.SaveAuthRequestInfo(ctx, info)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("SaveAuthRequestInfo_AlreadyExists", func(t *testing.T) {
		mock.ExpectSetArgs(key, data, redis.SetArgs{
			Mode: "NX",
			TTL:  AuthRequestTTL,
		}).SetVal("")

		err := store.SaveAuthRequestInfo(ctx, info)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("SaveAuthRequestInfo_RedisError", func(t *testing.T) {
		mock.ExpectSetArgs(key, data, redis.SetArgs{
			Mode: "NX",
			TTL:  AuthRequestTTL,
		}).SetErr(fmt.Errorf("redis down"))

		err := store.SaveAuthRequestInfo(ctx, info)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("DeleteAuthRequestInfo_Success", func(t *testing.T) {
		mock.ExpectDel(key).SetVal(1)

		err := store.DeleteAuthRequestInfo(ctx, state)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
