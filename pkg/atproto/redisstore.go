package atproto

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

var _ oauth.ClientAuthStore = &RedisStore{}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

func sessionKey(did syntax.DID, sessionID string) string {
	return fmt.Sprintf("oauth:session:%s/%s", did, sessionID)
}

func authRequestKey(state string) string {
	return fmt.Sprintf("oauth:authreq:%s", state)
}

func (s *RedisStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	data, err := s.client.Get(ctx, sessionKey(did, sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found: %s", did)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var sess oauth.ClientSessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &sess, nil
}

func (s *RedisStore) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := s.client.Set(ctx, sessionKey(sess.AccountDID, sess.SessionID), data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *RedisStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	if err := s.client.Del(ctx, sessionKey(did, sessionID)).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *RedisStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	data, err := s.client.Get(ctx, authRequestKey(state)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("request info not found: %s", state)
		}
		return nil, fmt.Errorf("failed to get auth request info: %w", err)
	}

	var req oauth.AuthRequestData
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth request info: %w", err)
	}

	return &req, nil
}

func (s *RedisStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	key := authRequestKey(info.State)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check auth request existence: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("auth request already saved for state %s", info.State)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request info: %w", err)
	}

	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save auth request info: %w", err)
	}

	return nil
}

func (s *RedisStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	if err := s.client.Del(ctx, authRequestKey(state)).Err(); err != nil {
		return fmt.Errorf("failed to delete auth request info: %w", err)
	}

	return nil
}
