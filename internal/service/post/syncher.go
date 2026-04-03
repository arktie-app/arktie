package post

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/atproto/syntax"

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

	did, err := syntax.ParseDID(accountDID)
	if err != nil {
		return fmt.Errorf("failed to parse DID %s: %w", accountDID, err)
	}

	sess, err := svc.client.OAuth.ResumeSession(msg.Context(), did, sessionID)
	if err != nil {
		return fmt.Errorf("failed to resume session for %s: %w", did, err)
	}

	apiClient := sess.APIClient()

	input := &agnostic.RepoPutRecord_Input{
		Repo:       apiClient.AccountDID.String(),
		Collection: "app.arktie.post",
		Rkey:       p.ID.String(),
		Record:     p.ToPDSRecord(),
	}

	var out agnostic.RepoPutRecord_Output
	if err := apiClient.Post(msg.Context(), "com.atproto.repo.putRecord", input, &out); err != nil {
		return fmt.Errorf("failed to create record on PDS: %w", err)
	}

	if out.Uri != "" {
		if err := svc.client.Ent.Post.UpdateOneID(p.ID).SetAtURL(out.Uri).Exec(msg.Context()); err != nil {
			slog.Error("failed to update post at_url", liblogs.ErrAttr(err), slog.String("post_id", p.ID.String()), slog.String("at_url", out.Uri))
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

	did, err := syntax.ParseDID(accountDID)
	if err != nil {
		return fmt.Errorf("failed to parse DID %s: %w", accountDID, err)
	}

	sess, err := svc.client.OAuth.ResumeSession(msg.Context(), did, sessionID)
	if err != nil {
		return fmt.Errorf("failed to resume session for %s: %w", did, err)
	}

	apiClient := sess.APIClient()

	input := &agnostic.RepoPutRecord_Input{
		Repo:       apiClient.AccountDID.String(),
		Collection: "app.arktie.post",
		Rkey:       p.ID.String(),
		Record:     p.ToPDSRecord(),
	}

	var out agnostic.RepoPutRecord_Output
	if err := apiClient.Post(msg.Context(), "com.atproto.repo.putRecord", input, &out); err != nil {
		return fmt.Errorf("failed to update record on PDS: %w", err)
	}

	if out.Uri != "" {
		if err := svc.client.Ent.Post.UpdateOneID(p.ID).SetAtURL(out.Uri).Exec(msg.Context()); err != nil {
			slog.Error("failed to update post at_url", liblogs.ErrAttr(err), slog.String("post_id", p.ID.String()), slog.String("at_url", out.Uri))
			return fmt.Errorf("failed to update at_url on posts: %w", err)
		}
	}

	return nil
}

func (svc *Service) Delete(msg *message.Message) error {
	rkey := string(msg.Payload)

	accountDID := msg.Metadata.Get("account_did")
	sessionID := msg.Metadata.Get("session_id")

	did, err := syntax.ParseDID(accountDID)
	if err != nil {
		return fmt.Errorf("failed to parse DID %s: %w", accountDID, err)
	}

	sess, err := svc.client.OAuth.ResumeSession(msg.Context(), did, sessionID)
	if err != nil {
		return fmt.Errorf("failed to resume session for %s: %w", did, err)
	}

	apiClient := sess.APIClient()

	input := &struct {
		Repo       string `json:"repo"`
		Collection string `json:"collection"`
		Rkey       string `json:"rkey"`
	}{
		Repo:       apiClient.AccountDID.String(),
		Collection: "app.arktie.post",
		Rkey:       rkey,
	}

	if err := apiClient.Post(msg.Context(), "com.atproto.repo.deleteRecord", input, nil); err != nil {
		return fmt.Errorf("failed to delete record on PDS: %w", err)
	}

	return nil
}
