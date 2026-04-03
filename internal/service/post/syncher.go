package post

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/ent"
	"arktie.org/internal/lib/liblogs"
)

func (svc *Service) Create(msg *message.Message) error {
	var p ent.Post
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal post.created event: %w", err)
	}

	accountDID := msg.Metadata.Get("account_did")
	sessionID := msg.Metadata.Get("session_id")

	uri, err := svc.pds.PutRecord(msg.Context(), accountDID, sessionID, "app.arktie.post", p.ID.String(), p.ToPDSRecord())
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

func (svc *Service) Update(msg *message.Message) error {
	var p ent.Post
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal post.updated event: %w", err)
	}

	accountDID := msg.Metadata.Get("account_did")
	sessionID := msg.Metadata.Get("session_id")

	uri, err := svc.pds.PutRecord(msg.Context(), accountDID, sessionID, "app.arktie.post", p.ID.String(), p.ToPDSRecord())
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

func (svc *Service) Delete(msg *message.Message) error {
	rkey := string(msg.Payload)

	accountDID := msg.Metadata.Get("account_did")
	sessionID := msg.Metadata.Get("session_id")

	return svc.pds.DeleteRecord(msg.Context(), accountDID, sessionID, "app.arktie.post", rkey)
}
