package post

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"arktie.org/ent"
	"arktie.org/ent/enttest"
	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
)

func createTestUser(t *testing.T, entClient *ent.Client, did string) *ent.User {
	t.Helper()
	return entClient.User.Create().
		SetAccountDid(did).
		SetEncryptionKey([]byte("test-key-0123456")).
		SaveX(context.Background())
}

func setupSyncTest(t *testing.T) (*client.Database, *ent.Client) {
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

	return &client.Database{Ent: entClient}, entClient
}

func newSyncMessage(t *testing.T, payload any, metadata map[string]string) *message.Message {
	t.Helper()

	var body []byte
	switch v := payload.(type) {
	case []byte:
		body = v
	case string:
		body = []byte(v)
	default:
		var err error
		body, err = json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
	}

	msg := message.NewMessage("", body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}
	return msg
}

// --- PushCreate ---

func TestPushCreate_Success(t *testing.T) {
	db, _ := setupSyncTest(t)

	content := "hello"
	now := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	// pre-create user and post in db
	u := createTestUser(t, db.Ent, "did:plc:test")
	p := db.Ent.Post.Create().
		SetID(testPostID).
		SetUserID(u.ID).
		SetMarkdownContent(content).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(context.Background())

	returnedURI := "at://did:plc:test/app.arktie.post/" + p.ID.String()

	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, accountDID, sessionID, col, rkey string, record *data.PDSRecord) (string, error) {
			if accountDID != "did:plc:test" {
				t.Errorf("expected accountDID did:plc:test, got %s", accountDID)
			}
			if sessionID != "sess-001" {
				t.Errorf("expected sessionID sess-001, got %s", sessionID)
			}
			if col != "app.arktie.post" {
				t.Errorf("expected collection app.arktie.post, got %s", col)
			}
			if rkey != p.ID.String() {
				t.Errorf("expected rkey %s, got %s", p.ID.String(), rkey)
			}
			return returnedURI, nil
		},
	}

	svc := NewSyncService(db, pds)

	msg := newSyncMessage(t, p, map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushCreate(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pds.PutRecordCalls()) != 1 {
		t.Fatalf("expected 1 PutRecord call, got %d", len(pds.PutRecordCalls()))
	}

	// verify at_url was updated in db
	updated := db.Ent.Post.GetX(context.Background(), p.ID)
	if updated.AtURL == nil || *updated.AtURL != returnedURI {
		t.Errorf("expected at_url %q, got %v", returnedURI, updated.AtURL)
	}
}

func TestPushCreate_EmptyURI(t *testing.T) {
	db, _ := setupSyncTest(t)

	content := "hello"
	now := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	u := createTestUser(t, db.Ent, "did:plc:test")
	p := db.Ent.Post.Create().
		SetID(testPostID).
		SetUserID(u.ID).
		SetMarkdownContent(content).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(context.Background())

	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ *data.PDSRecord) (string, error) {
			return "", nil
		},
	}

	svc := NewSyncService(db, pds)
	msg := newSyncMessage(t, p, map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushCreate(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// at_url should remain nil
	updated := db.Ent.Post.GetX(context.Background(), p.ID)
	if updated.AtURL != nil {
		t.Errorf("expected at_url nil, got %v", updated.AtURL)
	}
}

func TestPushCreate_PutRecordError(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ *data.PDSRecord) (string, error) {
			return "", errors.New("pds error")
		},
	}

	svc := NewSyncService(nil, pds)

	content := "test"
	p := newPost(testPostID, testUserID, &content)

	msg := newSyncMessage(t, p, map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushCreate(msg); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPushCreate_InvalidPayload(t *testing.T) {
	svc := NewSyncService(nil, nil)

	msg := newSyncMessage(t, []byte("invalid json"), nil)
	if err := svc.PushCreate(msg); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

// --- PushUpdate ---

func TestPushUpdate_Success(t *testing.T) {
	db, _ := setupSyncTest(t)

	content := "updated"
	now := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	u := createTestUser(t, db.Ent, "did:plc:test")
	p := db.Ent.Post.Create().
		SetID(testPostID).
		SetUserID(u.ID).
		SetMarkdownContent(content).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(context.Background())

	returnedURI := "at://did:plc:test/app.arktie.post/" + p.ID.String()

	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ *data.PDSRecord) (string, error) {
			return returnedURI, nil
		},
	}

	svc := NewSyncService(db, pds)
	msg := newSyncMessage(t, p, map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushUpdate(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := db.Ent.Post.GetX(context.Background(), p.ID)
	if updated.AtURL == nil || *updated.AtURL != returnedURI {
		t.Errorf("expected at_url %q, got %v", returnedURI, updated.AtURL)
	}
}

func TestPushUpdate_PutRecordError(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ *data.PDSRecord) (string, error) {
			return "", errors.New("pds error")
		},
	}

	svc := NewSyncService(nil, pds)
	p := newPost(testPostID, testUserID, nil)

	msg := newSyncMessage(t, p, map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushUpdate(msg); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- PushDelete ---

func TestPushDelete_Success(t *testing.T) {
	pds := &PDSRecordAPIMock{
		DeleteRecordFunc: func(_ context.Context, accountDID, sessionID, col, rkey string) error {
			if accountDID != "did:plc:test" {
				t.Errorf("expected accountDID did:plc:test, got %s", accountDID)
			}
			if sessionID != "sess-001" {
				t.Errorf("expected sessionID sess-001, got %s", sessionID)
			}
			if col != "app.arktie.post" {
				t.Errorf("expected collection app.arktie.post, got %s", col)
			}
			if rkey != testPostID.String() {
				t.Errorf("expected rkey %s, got %s", testPostID, rkey)
			}
			return nil
		},
	}

	svc := NewSyncService(nil, pds)
	msg := newSyncMessage(t, testPostID.String(), map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushDelete(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushDelete_Error(t *testing.T) {
	pds := &PDSRecordAPIMock{
		DeleteRecordFunc: func(_ context.Context, _, _, _, _ string) error {
			return errors.New("delete failed")
		},
	}

	svc := NewSyncService(nil, pds)
	msg := newSyncMessage(t, testPostID.String(), map[string]string{
		"account_did": "did:plc:test",
		"session_id":  "sess-001",
	})

	if err := svc.PushDelete(msg); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- PullCreate ---

func TestPullCreate_Success(t *testing.T) {
	db, _ := setupSyncTest(t)

	// create a user first
	createTestUser(t, db.Ent, "did:plc:pull-test")

	postID := uuid.Must(uuid.NewV7())
	content := "pulled content"
	now := syntax.DatetimeNow()
	record := &data.PDSRecord{
		Content:       &content,
		CreatedAt:     now,
		UpdatedAt:     now,
		PublishedAt:   now,
		PublishedFrom: "https://example.com",
	}

	svc := NewSyncService(db, nil)
	msg := newSyncMessage(t, record, map[string]string{
		"repo": "did:plc:pull-test",
		"path": "app.arktie.post/" + postID.String(),
	})

	if err := svc.PullCreate(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	created := db.Ent.Post.GetX(context.Background(), postID)
	if created.MarkdownContent == nil || *created.MarkdownContent != content {
		t.Errorf("expected content %q, got %v", content, created.MarkdownContent)
	}
	if created.AtURL == nil || *created.AtURL != "at://did:plc:pull-test/app.arktie.post/"+postID.String() {
		t.Errorf("unexpected at_url: %v", created.AtURL)
	}
	if created.PublishFrom == nil || *created.PublishFrom != "https://example.com" {
		t.Errorf("expected publish_from https://example.com, got %v", created.PublishFrom)
	}
}

func TestPullCreate_UserNotFound(t *testing.T) {
	db, _ := setupSyncTest(t)

	postID := uuid.Must(uuid.NewV7())
	now := syntax.DatetimeNow()
	record := &data.PDSRecord{CreatedAt: now, UpdatedAt: now, PublishedAt: now, PublishedFrom: "https://example.com"}

	svc := NewSyncService(db, nil)
	msg := newSyncMessage(t, record, map[string]string{
		"repo": "did:plc:unknown",
		"path": "app.arktie.post/" + postID.String(),
	})

	if err := svc.PullCreate(msg); err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
}

func TestPullCreate_InvalidRkey(t *testing.T) {
	db, _ := setupSyncTest(t)

	now := syntax.DatetimeNow()
	record := &data.PDSRecord{CreatedAt: now, UpdatedAt: now, PublishedAt: now, PublishedFrom: "https://example.com"}

	svc := NewSyncService(db, nil)
	msg := newSyncMessage(t, record, map[string]string{
		"repo": "did:plc:test",
		"path": "app.arktie.post/not-a-uuid",
	})

	if err := svc.PullCreate(msg); err == nil {
		t.Fatal("expected error for invalid rkey, got nil")
	}
}

func TestPullCreate_InvalidPath(t *testing.T) {
	svc := NewSyncService(nil, nil)

	record := &data.PDSRecord{}
	msg := newSyncMessage(t, record, map[string]string{
		"repo": "did:plc:test",
		"path": "no-slash",
	})

	if err := svc.PullCreate(msg); err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestPullCreate_InvalidPayload(t *testing.T) {
	svc := NewSyncService(nil, nil)

	msg := newSyncMessage(t, []byte("bad json"), map[string]string{
		"repo": "did:plc:test",
		"path": "app.arktie.post/" + uuid.Must(uuid.NewV7()).String(),
	})

	if err := svc.PullCreate(msg); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

// --- PullUpdate ---

func TestPullUpdate_Success(t *testing.T) {
	db, _ := setupSyncTest(t)

	u := createTestUser(t, db.Ent, "did:plc:test")
	postID := uuid.Must(uuid.NewV7())
	origContent := "original"
	now := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	db.Ent.Post.Create().
		SetID(postID).
		SetUserID(u.ID).
		SetMarkdownContent(origContent).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(context.Background())

	updatedContent := "updated from firehose"
	updatedAt := syntax.DatetimeNow()
	record := &data.PDSRecord{
		Content:     &updatedContent,
		CreatedAt:   updatedAt,
		UpdatedAt:   updatedAt,
		PublishedAt: updatedAt,
	}

	svc := NewSyncService(db, nil)
	msg := newSyncMessage(t, record, map[string]string{
		"path": "app.arktie.post/" + postID.String(),
	})

	if err := svc.PullUpdate(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := db.Ent.Post.GetX(context.Background(), postID)
	if updated.MarkdownContent == nil || *updated.MarkdownContent != updatedContent {
		t.Errorf("expected content %q, got %v", updatedContent, updated.MarkdownContent)
	}
}

func TestPullUpdate_PostNotFound(t *testing.T) {
	db, _ := setupSyncTest(t)

	postID := uuid.Must(uuid.NewV7())
	now := syntax.DatetimeNow()
	record := &data.PDSRecord{CreatedAt: now, UpdatedAt: now, PublishedAt: now}

	svc := NewSyncService(db, nil)
	msg := newSyncMessage(t, record, map[string]string{
		"path": "app.arktie.post/" + postID.String(),
	})

	if err := svc.PullUpdate(msg); err == nil {
		t.Fatal("expected error for missing post, got nil")
	}
}

func TestPullUpdate_InvalidPayload(t *testing.T) {
	svc := NewSyncService(nil, nil)

	msg := newSyncMessage(t, []byte("bad json"), map[string]string{
		"path": "app.arktie.post/" + uuid.Must(uuid.NewV7()).String(),
	})

	if err := svc.PullUpdate(msg); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

// --- PullDelete ---

func TestPullDelete_Success(t *testing.T) {
	db, _ := setupSyncTest(t)

	u := createTestUser(t, db.Ent, "did:plc:test")
	postID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	db.Ent.Post.Create().
		SetID(postID).
		SetUserID(u.ID).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SaveX(context.Background())

	svc := NewSyncService(db, nil)

	path := "app.arktie.post/" + postID.String()
	msg := newSyncMessage(t, path, map[string]string{})

	if err := svc.PullDelete(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := db.Ent.Post.Query().CountX(context.Background())
	if count != 0 {
		t.Errorf("expected 0 posts, got %d", count)
	}
}

func TestPullDelete_InvalidRkey(t *testing.T) {
	svc := NewSyncService(nil, nil)

	msg := newSyncMessage(t, "app.arktie.post/not-a-uuid", nil)

	if err := svc.PullDelete(msg); err == nil {
		t.Fatal("expected error for invalid rkey, got nil")
	}
}

func TestPullDelete_InvalidPath(t *testing.T) {
	svc := NewSyncService(nil, nil)

	msg := newSyncMessage(t, "no-slash", nil)

	if err := svc.PullDelete(msg); err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

// --- rkeyFromPath ---

func TestRkeyFromPath(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{"app.arktie.post/abc123", "abc123", false},
		{"collection/rkey", "rkey", false},
		{"no-slash", "", true},
		{"trailing/", "", true},
	}

	for _, tt := range tests {
		got, err := rkeyFromPath(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("rkeyFromPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("rkeyFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
