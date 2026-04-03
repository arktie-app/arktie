package post

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"arktie.org/internal/data/client"
)

// PDSRecord handles PDS record operations via OAuth sessions.
type PDSRecord struct {
	oauth *client.OAuth
}

func NewPDSRecord(oauth *client.OAuth) *PDSRecord {
	return &PDSRecord{oauth: oauth}
}

func (p *PDSRecord) PutRecord(ctx context.Context, accountDID, sessionID, collection, rkey string, record map[string]any) (string, error) {
	did, err := syntax.ParseDID(accountDID)
	if err != nil {
		return "", fmt.Errorf("failed to parse DID %s: %w", accountDID, err)
	}

	sess, err := p.oauth.ATProto.ResumeSession(ctx, did, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to resume session for %s: %w", did, err)
	}

	apiClient := sess.APIClient()

	input := &agnostic.RepoPutRecord_Input{
		Repo:       apiClient.AccountDID.String(),
		Collection: collection,
		Rkey:       rkey,
		Record:     record,
	}

	var out agnostic.RepoPutRecord_Output
	if err := apiClient.Post(ctx, "com.atproto.repo.putRecord", input, &out); err != nil {
		return "", fmt.Errorf("failed to put record on PDS: %w", err)
	}

	return out.Uri, nil
}

func (p *PDSRecord) DeleteRecord(ctx context.Context, accountDID, sessionID, collection, rkey string) error {
	did, err := syntax.ParseDID(accountDID)
	if err != nil {
		return fmt.Errorf("failed to parse DID %s: %w", accountDID, err)
	}

	sess, err := p.oauth.ATProto.ResumeSession(ctx, did, sessionID)
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
		Collection: collection,
		Rkey:       rkey,
	}

	if err := apiClient.Post(ctx, "com.atproto.repo.deleteRecord", input, nil); err != nil {
		return fmt.Errorf("failed to delete record on PDS: %w", err)
	}

	return nil
}
