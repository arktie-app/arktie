package post

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"

	_ "modernc.org/sqlite"

	"arktie.org/ent"
	"arktie.org/ent/enttest"
	"arktie.org/internal/data"
)

func setupSyncherTest(t *testing.T, pds *PDSRecordAPIMock) (*Service, *ent.Client) {
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

	svc := &Service{
		client: &data.Client{Ent: entClient},
		pds:    pds,
	}

	return svc, entClient
}

func createTestUser(t *testing.T, entClient *ent.Client) *ent.User {
	t.Helper()
	u, err := entClient.User.Create().
		SetAccountDid("did:plc:test123").
		SetEncryptionKey([]byte("01234567890123456789012345678901")).
		Save(context.Background())
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

func createTestPost(t *testing.T, entClient *ent.Client, userID uuid.UUID, content *string) *ent.Post {
	t.Helper()
	builder := entClient.Post.Create().SetUserID(userID)
	if content != nil {
		builder.SetMarkdownContent(*content)
	}
	p, err := builder.Save(context.Background())
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}
	return p
}

func newSyncherMessage(t *testing.T, payload []byte, accountDID, sessionID string) *message.Message {
	t.Helper()
	msg := message.NewMessage(uuid.Must(uuid.NewV7()).String(), payload)
	msg.Metadata.Set("account_did", accountDID)
	msg.Metadata.Set("session_id", sessionID)
	msg.SetContext(context.Background())
	return msg
}

// --- Create ---

func TestSyncher_Create_Success(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, accountDID, sessionID, collection, rkey string, _ map[string]any) (string, error) {
			if collection != "app.arktie.post" {
				t.Errorf("expected collection app.arktie.post, got %s", collection)
			}
			if accountDID != "did:plc:test123" {
				t.Errorf("expected accountDID did:plc:test123, got %s", accountDID)
			}
			return "at://did:plc:test123/app.arktie.post/" + rkey, nil
		},
	}

	svc, entClient := setupSyncherTest(t, pds)
	user := createTestUser(t, entClient)
	content := "hello"
	post := createTestPost(t, entClient, user.ID, &content)

	payload, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	msg := newSyncherMessage(t, payload, "did:plc:test123", "sess-001")

	if err := svc.Create(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := entClient.Post.Get(context.Background(), post.ID)
	if err != nil {
		t.Fatal(err)
	}
	expectedURL := "at://did:plc:test123/app.arktie.post/" + post.ID.String()
	if updated.AtURL == nil || *updated.AtURL != expectedURL {
		t.Errorf("expected at_url %s, got %v", expectedURL, updated.AtURL)
	}

	if len(pds.PutRecordCalls()) != 1 {
		t.Errorf("expected 1 PutRecord call, got %d", len(pds.PutRecordCalls()))
	}
}

func TestSyncher_Create_PutRecordError(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ map[string]any) (string, error) {
			return "", errors.New("pds error")
		},
	}

	svc, entClient := setupSyncherTest(t, pds)
	user := createTestUser(t, entClient)
	post := createTestPost(t, entClient, user.ID, nil)

	payload, _ := json.Marshal(post)
	msg := newSyncherMessage(t, payload, "did:plc:test123", "sess-001")

	err := svc.Create(msg)
	if err == nil || err.Error() != "pds error" {
		t.Fatalf("expected pds error, got %v", err)
	}
}

func TestSyncher_Create_InvalidPayload(t *testing.T) {
	svc, _ := setupSyncherTest(t, &PDSRecordAPIMock{})

	msg := newSyncherMessage(t, []byte("not json"), "did:plc:test123", "sess-001")

	err := svc.Create(msg)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestSyncher_Create_EmptyURI_SkipsUpdate(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ map[string]any) (string, error) {
			return "", nil
		},
	}

	svc, entClient := setupSyncherTest(t, pds)
	user := createTestUser(t, entClient)
	post := createTestPost(t, entClient, user.ID, nil)

	payload, _ := json.Marshal(post)
	msg := newSyncherMessage(t, payload, "did:plc:test123", "sess-001")

	if err := svc.Create(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := entClient.Post.Get(context.Background(), post.ID)
	if updated.AtURL != nil {
		t.Errorf("expected at_url to remain nil, got %v", updated.AtURL)
	}
}

// --- Update ---

func TestSyncher_Update_Success(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, collection, rkey string, _ map[string]any) (string, error) {
			if collection != "app.arktie.post" {
				t.Errorf("expected collection app.arktie.post, got %s", collection)
			}
			return "at://did:plc:test123/app.arktie.post/" + rkey, nil
		},
	}

	svc, entClient := setupSyncherTest(t, pds)
	user := createTestUser(t, entClient)
	content := "original"
	post := createTestPost(t, entClient, user.ID, &content)

	// Simulate update: change content in the serialized post
	post.MarkdownContent = strPtr("updated content")
	post.UpdatedAt = time.Now()
	payload, _ := json.Marshal(post)
	msg := newSyncherMessage(t, payload, "did:plc:test123", "sess-001")

	if err := svc.Update(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := entClient.Post.Get(context.Background(), post.ID)
	if updated.AtURL == nil {
		t.Fatal("expected at_url to be set")
	}

	if len(pds.PutRecordCalls()) != 1 {
		t.Errorf("expected 1 PutRecord call, got %d", len(pds.PutRecordCalls()))
	}
}

func TestSyncher_Update_PutRecordError(t *testing.T) {
	pds := &PDSRecordAPIMock{
		PutRecordFunc: func(_ context.Context, _, _, _, _ string, _ map[string]any) (string, error) {
			return "", errors.New("pds error")
		},
	}

	svc, entClient := setupSyncherTest(t, pds)
	user := createTestUser(t, entClient)
	post := createTestPost(t, entClient, user.ID, nil)

	payload, _ := json.Marshal(post)
	msg := newSyncherMessage(t, payload, "did:plc:test123", "sess-001")

	err := svc.Update(msg)
	if err == nil || err.Error() != "pds error" {
		t.Fatalf("expected pds error, got %v", err)
	}
}

// --- Delete ---

func TestSyncher_Delete_Success(t *testing.T) {
	postID := uuid.Must(uuid.NewV7())

	pds := &PDSRecordAPIMock{
		DeleteRecordFunc: func(_ context.Context, accountDID, sessionID, collection, rkey string) error {
			if collection != "app.arktie.post" {
				t.Errorf("expected collection app.arktie.post, got %s", collection)
			}
			if rkey != postID.String() {
				t.Errorf("expected rkey %s, got %s", postID, rkey)
			}
			if accountDID != "did:plc:test123" {
				t.Errorf("expected accountDID did:plc:test123, got %s", accountDID)
			}
			return nil
		},
	}

	svc, _ := setupSyncherTest(t, pds)

	msg := newSyncherMessage(t, []byte(postID.String()), "did:plc:test123", "sess-001")

	if err := svc.Delete(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pds.DeleteRecordCalls()) != 1 {
		t.Errorf("expected 1 DeleteRecord call, got %d", len(pds.DeleteRecordCalls()))
	}
}

func TestSyncher_Delete_PdsError(t *testing.T) {
	pds := &PDSRecordAPIMock{
		DeleteRecordFunc: func(_ context.Context, _, _, _, _ string) error {
			return errors.New("pds error")
		},
	}

	svc, _ := setupSyncherTest(t, pds)

	msg := newSyncherMessage(t, []byte(uuid.Must(uuid.NewV7()).String()), "did:plc:test123", "sess-001")

	err := svc.Delete(msg)
	if err == nil || err.Error() != "pds error" {
		t.Fatalf("expected pds error, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
