package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"arktie.org/ent"
	entpost "arktie.org/ent/post"
	"arktie.org/ent/user"
	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/liblogs"
	ucpost "arktie.org/internal/usecase/post"
)

type SyncService struct {
	db *client.Database

	pds PDSRecordAPI
}

func NewSyncService(db *client.Database, pds PDSRecordAPI) *SyncService {
	return &SyncService{
		db:  db,
		pds: pds,
	}
}

func (svc *SyncService) PushCreate(msg *message.Message) error {
	return svc.pushUpsert(msg)
}

func (svc *SyncService) PushUpdate(msg *message.Message) error {
	return svc.pushUpsert(msg)
}

func (svc *SyncService) pushUpsert(msg *message.Message) error {
	var p ent.Post
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal post event: %w", err)
	}

	accountDID := msg.Metadata.Get("account_did")
	sessionID := msg.Metadata.Get("session_id")

	uri, err := svc.pds.PutRecord(
		msg.Context(),
		accountDID, sessionID,
		"app.arktie.post",
		p.ID.String(),
		p.ToPDSRecord(),
	)
	if err != nil {
		return err
	}

	if uri != "" {
		if err := svc.db.Ent.Post.UpdateOneID(p.ID).SetAtURL(uri).Exec(msg.Context()); err != nil {
			slog.Error("failed to update post at_url", liblogs.ErrAttr(err), slog.String("post_id", p.ID.String()), slog.String("at_url", uri))
			return fmt.Errorf("failed to update at_url on posts: %w", err)
		}
	}

	return nil
}

func (svc *SyncService) PushDelete(msg *message.Message) error {
	rkey := string(msg.Payload)

	accountDID := msg.Metadata.Get("account_did")
	sessionID := msg.Metadata.Get("session_id")

	return svc.pds.DeleteRecord(msg.Context(), accountDID, sessionID, "app.arktie.post", rkey)
}

func (svc *SyncService) PullCreate(msg *message.Message) error {
	var record data.PDSRecord
	if err := json.Unmarshal(msg.Payload, &record); err != nil {
		return fmt.Errorf("failed to unmarshal firehose post record: %w", err)
	}

	repo := msg.Metadata.Get("repo")
	path := msg.Metadata.Get("path")

	rkey, err := rkeyFromPath(path)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(rkey)
	if err != nil {
		return fmt.Errorf("failed to parse rkey %q as UUID: %w", rkey, err)
	}

	ctx := msg.Context()

	u, err := svc.db.Ent.User.Query().Where(user.AccountDid(repo)).Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to find user for DID %s: %w", repo, err)
	}

	atURL := fmt.Sprintf("at://%s/%s/%s", repo, collection, rkey)

	create := svc.db.Ent.Post.Create().
		SetID(postID).
		SetUserID(u.ID).
		SetNillableMarkdownContent(record.Content).
		SetAtURL(atURL).
		SetPublishFrom(record.PublishedFrom)

	if t, err := time.Parse(syntax.AtprotoDatetimeLayout, string(record.CreatedAt)); err == nil {
		create.SetCreatedAt(t)
	}
	if t, err := time.Parse(syntax.AtprotoDatetimeLayout, string(record.UpdatedAt)); err == nil {
		create.SetUpdatedAt(t)
	}

	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("failed to create post from firehose: %w", err)
	}

	return nil
}

func (svc *SyncService) PullUpdate(msg *message.Message) error {
	var record data.PDSRecord
	if err := json.Unmarshal(msg.Payload, &record); err != nil {
		return fmt.Errorf("failed to unmarshal firehose post record: %w", err)
	}

	path := msg.Metadata.Get("path")

	rkey, err := rkeyFromPath(path)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(rkey)
	if err != nil {
		return fmt.Errorf("failed to parse rkey %q as UUID: %w", rkey, err)
	}

	ctx := msg.Context()

	update := svc.db.Ent.Post.UpdateOneID(postID).
		SetNillableMarkdownContent(record.Content)

	if t, err := time.Parse(syntax.AtprotoDatetimeLayout, string(record.UpdatedAt)); err == nil {
		update.SetUpdatedAt(t)
	}

	if err := update.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update post from firehose: %w", err)
	}

	return nil
}

func (svc *SyncService) PullDelete(msg *message.Message) error {
	path := string(msg.Payload)

	rkey, err := rkeyFromPath(path)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(rkey)
	if err != nil {
		return fmt.Errorf("failed to parse rkey %q as UUID: %w", rkey, err)
	}

	ctx := msg.Context()

	if _, err := svc.db.Ent.Post.Delete().Where(entpost.ID(postID)).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete post from firehose: %w", err)
	}

	return nil
}

func rkeyFromPath(path string) (string, error) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid path format: %s", path)
	}
	return parts[1], nil
}

//go:generate go tool moq -rm -out mock_pds.go . PDSRecordAPI

// PDSRecordAPI abstracts PDS record operations for testability.
type PDSRecordAPI interface {
	PutRecord(ctx context.Context, accountDID, sessionID, collection, rkey string, record *data.PDSRecord) (uri string, err error)
	DeleteRecord(ctx context.Context, accountDID, sessionID, collection, rkey string) error
}

var _ PDSRecordAPI = &ucpost.PDSRecord{}
