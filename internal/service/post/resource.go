package post

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"arktie.org/ent"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/liblogs"
	postv1 "arktie.org/internal/pb/post/v1"
	"arktie.org/internal/server/middleware"
	ucpost "arktie.org/internal/usecase/post"
)

type ResourceService struct {
	postv1.UnimplementedPostServiceServer

	event    *client.Event
	resource PostResource
}

var _ postv1.PostServiceServer = &ResourceService{}

func NewResourceService(event *client.Event, resource PostResource) *ResourceService {
	return &ResourceService{
		event:    event,
		resource: resource,
	}
}

func (svc *ResourceService) CreatePost(ctx context.Context, req *postv1.CreatePostRequest) (*postv1.CreatePostResponse, error) {
	claim := middleware.UserFromContext(ctx)
	if claim == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	userID, err := uuid.Parse(claim.Subject)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user")
	}

	p, err := svc.resource.Create(ctx, userID, req.MarkdownContent)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create post")
	}

	if payload, err := json.Marshal(p); err == nil {
		msg := message.NewMessage(uuid.Must(uuid.NewV7()).String(), payload)
		msg.Metadata.Set("account_did", claim.ATProto.Session.AccountDID.String())
		msg.Metadata.Set("session_id", claim.ATProto.Session.SessionID)
		if err := svc.event.Publisher.Publish("post.created", msg); err != nil {
			slog.ErrorContext(ctx, "failed to publish post.create", liblogs.ErrAttr(err))
			return nil, status.Error(codes.Internal, "failed to publish post.create")
		}
	} else {
		return nil, status.Error(codes.Internal, "failed to encode json")
	}

	return &postv1.CreatePostResponse{Post: p.ToProto()}, nil
}

func (svc *ResourceService) GetPost(ctx context.Context, req *postv1.GetPostRequest) (*postv1.GetPostResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post id")
	}

	p, err := svc.resource.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		return nil, status.Error(codes.Internal, "failed to get post")
	}

	return &postv1.GetPostResponse{Post: p.ToProto()}, nil
}

func (svc *ResourceService) UpdatePost(ctx context.Context, req *postv1.UpdatePostRequest) (*postv1.UpdatePostResponse, error) {
	claim := middleware.UserFromContext(ctx)
	if claim == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	userID, err := uuid.Parse(claim.Subject)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post id")
	}

	updated, err := svc.resource.Update(ctx, id, userID, req.MarkdownContent)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		if err == ucpost.ErrForbidden {
			return nil, status.Error(codes.PermissionDenied, "you are not the owner of this post")
		}
		return nil, status.Error(codes.Internal, "failed to update post")
	}

	if payload, err := json.Marshal(updated); err == nil {
		msg := message.NewMessage(uuid.Must(uuid.NewV7()).String(), payload)
		msg.Metadata.Set("account_did", claim.ATProto.Session.AccountDID.String())
		msg.Metadata.Set("session_id", claim.ATProto.Session.SessionID)
		if err := svc.event.Publisher.Publish("post.updated", msg); err != nil {
			slog.ErrorContext(ctx, "failed to publish post.update", liblogs.ErrAttr(err))
			return nil, status.Error(codes.Internal, "failed to publish post.update")
		}
	} else {
		return nil, status.Error(codes.Internal, "failed to encode json")
	}

	return &postv1.UpdatePostResponse{Post: updated.ToProto()}, nil
}

func (svc *ResourceService) DeletePost(ctx context.Context, req *postv1.DeletePostRequest) (*postv1.DeletePostResponse, error) {
	claim := middleware.UserFromContext(ctx)
	if claim == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	userID, err := uuid.Parse(claim.Subject)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post id")
	}

	if err := svc.resource.Delete(ctx, id, userID); err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete post")
	}

	{
		msg := message.NewMessage(uuid.Must(uuid.NewV7()).String(), message.Payload(id.String()))
		msg.Metadata.Set("account_did", claim.ATProto.Session.AccountDID.String())
		msg.Metadata.Set("session_id", claim.ATProto.Session.SessionID)
		if err := svc.event.Publisher.Publish("post.deleted", msg); err != nil {
			slog.ErrorContext(ctx, "failed to publish post.deleted", liblogs.ErrAttr(err))
			return nil, status.Error(codes.Internal, "failed to publish post.deleted")
		}
	}

	return &postv1.DeletePostResponse{}, nil
}

//go:generate go tool moq -rm -out mock_post_resource.go . PostResource
type PostResource interface {
	Create(ctx context.Context, userID uuid.UUID, markdownContent *string) (*ent.Post, error)
	Get(ctx context.Context, id uuid.UUID) (*ent.Post, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, markdownContent *string) (*ent.Post, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

var _ PostResource = &ucpost.Usecase{}
