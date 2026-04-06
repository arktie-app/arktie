package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/ent"
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
	var p ent.Post
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal post.created event: %w", err)
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

func (svc *SyncService) PushUpdate(msg *message.Message) error {
	var p ent.Post
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal post.updated event: %w", err)
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
	return nil
}

func (svc *SyncService) PullUpdate(msg *message.Message) error {
	return nil
}

func (svc *SyncService) PullDelete(msg *message.Message) error {
	return nil
}

//go:generate go tool moq -rm -out mock_pds.go . PDSRecordAPI

// PDSRecordAPI abstracts PDS record operations for testability.
type PDSRecordAPI interface {
	PutRecord(ctx context.Context, accountDID, sessionID, collection, rkey string, record *data.PDSRecord) (uri string, err error)
	DeleteRecord(ctx context.Context, accountDID, sessionID, collection, rkey string) error
}

var _ PDSRecordAPI = &ucpost.PDSRecord{}
