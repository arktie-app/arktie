package post

import (
	"context"

	"github.com/google/uuid"

	"arktie.org/ent"
	"arktie.org/internal/data"
	postv1 "arktie.org/internal/pb/post/v1"
	ucpost "arktie.org/internal/usecase/post"
)

type Service struct {
	postv1.UnimplementedPostServiceServer

	resource PostResource
	client   *data.Client
}

func NewService(resource PostResource, client *data.Client) *Service {
	return &Service{
		resource: resource,
		client:   client,
	}
}

type PostResource interface {
	Create(ctx context.Context, userID uuid.UUID, markdownContent *string) (*ent.Post, error)
	Get(ctx context.Context, id uuid.UUID) (*ent.Post, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, markdownContent *string) (*ent.Post, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

var _ PostResource = &ucpost.Usecase{}
