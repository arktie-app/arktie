package post

import (
	"context"

	"github.com/google/uuid"

	"arktie.org/ent"
	"arktie.org/internal/data"
	"arktie.org/internal/data/client"
	postv1 "arktie.org/internal/pb/post/v1"
	ucpost "arktie.org/internal/usecase/post"
)

type Service struct {
	postv1.UnimplementedPostServiceServer

	cfg   *data.Config
	db    *client.Database
	event *client.Event

	resource PostResource
	pds      PDSRecordAPI
}

func NewService(
	cfg *data.Config,
	db *client.Database,
	event *client.Event,

	resource PostResource,
	pds PDSRecordAPI,
) *Service {
	return &Service{
		cfg:   cfg,
		db:    db,
		event: event,

		resource: resource,
		pds:      pds,
	}
}

//go:generate go tool moq -rm -out mock_post_resource.go . PostResource
type PostResource interface {
	Create(ctx context.Context, userID uuid.UUID, markdownContent *string) (*ent.Post, error)
	Get(ctx context.Context, id uuid.UUID) (*ent.Post, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, markdownContent *string) (*ent.Post, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

var _ PostResource = &ucpost.Usecase{}

//go:generate go tool moq -rm -out mock_pds.go . PDSRecordAPI

// PDSRecordAPI abstracts PDS record operations for testability.
type PDSRecordAPI interface {
	PutRecord(ctx context.Context, accountDID, sessionID, collection, rkey string, record map[string]any) (uri string, err error)
	DeleteRecord(ctx context.Context, accountDID, sessionID, collection, rkey string) error
}

var _ PDSRecordAPI = &ucpost.PDSRecord{}
