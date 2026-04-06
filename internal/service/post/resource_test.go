package post

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"arktie.org/ent"
	"arktie.org/internal/data/client"
	"arktie.org/internal/lib/libjwt"
	postv1 "arktie.org/internal/pb/post/v1"
	"arktie.org/internal/server/middleware"
	ucpost "arktie.org/internal/usecase/post"
)

var (
	testUserID = uuid.Must(uuid.NewV7())
	testPostID = uuid.Must(uuid.NewV7())
)

func testClaim() *libjwt.Claim {
	return &libjwt.Claim{
		ATProto: libjwt.ATProto{
			Session: libjwt.ATProtoSession{
				AccountDID: syntax.DID("did:plc:test123"),
				SessionID:  "sess-001",
			},
		},
	}
}

func authedCtx(claim *libjwt.Claim) context.Context {
	claim.Subject = testUserID.String()
	return middleware.ContextWithUser(context.Background(), claim)
}

func newTestService(resource *PostResourceMock, publisher *client.MessagePublisherMock) *ResourceService {
	return NewResourceService(
		&client.Event{
			Publisher: publisher,
		},
		resource,
	)
}

func newPost(id, userID uuid.UUID, content *string) *ent.Post {
	now := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	return &ent.Post{
		ID:              id,
		UserID:          userID,
		MarkdownContent: content,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// --- CreatePost ---

func TestCreatePost_Success(t *testing.T) {
	content := "hello world"
	post := newPost(testPostID, testUserID, &content)

	resource := &PostResourceMock{
		CreateFunc: func(_ context.Context, userID uuid.UUID, mc *string) (*ent.Post, error) {
			if userID != testUserID {
				t.Errorf("expected userID %s, got %s", testUserID, userID)
			}
			return post, nil
		},
	}
	publisher := &client.MessagePublisherMock{
		PublishFunc: func(topic string, msgs ...*message.Message) error {
			if topic != "post.created" {
				t.Errorf("expected topic post.created, got %s", topic)
			}
			return nil
		},
	}

	svc := newTestService(resource, publisher)
	resp, err := svc.CreatePost(authedCtx(testClaim()), &postv1.CreatePostRequest{
		MarkdownContent: &content,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Post.Id != testPostID.String() {
		t.Errorf("expected post id %s, got %s", testPostID, resp.Post.Id)
	}
	if len(resource.CreateCalls()) != 1 {
		t.Errorf("expected 1 Create call, got %d", len(resource.CreateCalls()))
	}
	if len(publisher.PublishCalls()) != 1 {
		t.Errorf("expected 1 Publish call, got %d", len(publisher.PublishCalls()))
	}
}

func TestCreatePost_Unauthenticated(t *testing.T) {
	svc := newTestService(&PostResourceMock{}, &client.MessagePublisherMock{})

	_, err := svc.CreatePost(context.Background(), &postv1.CreatePostRequest{})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestCreatePost_ResourceError(t *testing.T) {
	resource := &PostResourceMock{
		CreateFunc: func(_ context.Context, _ uuid.UUID, _ *string) (*ent.Post, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(resource, &client.MessagePublisherMock{})

	_, err := svc.CreatePost(authedCtx(testClaim()), &postv1.CreatePostRequest{})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestCreatePost_PublishError(t *testing.T) {
	content := "test"
	resource := &PostResourceMock{
		CreateFunc: func(_ context.Context, _ uuid.UUID, _ *string) (*ent.Post, error) {
			return newPost(testPostID, testUserID, &content), nil
		},
	}
	publisher := &client.MessagePublisherMock{
		PublishFunc: func(_ string, _ ...*message.Message) error {
			return errors.New("publish failed")
		},
	}
	svc := newTestService(resource, publisher)

	_, err := svc.CreatePost(authedCtx(testClaim()), &postv1.CreatePostRequest{
		MarkdownContent: &content,
	})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

// --- GetPost ---

func TestGetPost_Success(t *testing.T) {
	content := "hello"
	post := newPost(testPostID, testUserID, &content)

	resource := &PostResourceMock{
		GetFunc: func(_ context.Context, id uuid.UUID) (*ent.Post, error) {
			if id != testPostID {
				t.Errorf("expected id %s, got %s", testPostID, id)
			}
			return post, nil
		},
	}
	svc := newTestService(resource, &client.MessagePublisherMock{})

	resp, err := svc.GetPost(context.Background(), &postv1.GetPostRequest{Id: testPostID.String()})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Post.Id != testPostID.String() {
		t.Errorf("expected post id %s, got %s", testPostID, resp.Post.Id)
	}
}

func TestGetPost_InvalidID(t *testing.T) {
	svc := newTestService(&PostResourceMock{}, &client.MessagePublisherMock{})

	_, err := svc.GetPost(context.Background(), &postv1.GetPostRequest{Id: "not-a-uuid"})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	resource := &PostResourceMock{
		GetFunc: func(_ context.Context, _ uuid.UUID) (*ent.Post, error) {
			return nil, &ent.NotFoundError{}
		},
	}
	svc := newTestService(resource, &client.MessagePublisherMock{})

	_, err := svc.GetPost(context.Background(), &postv1.GetPostRequest{Id: testPostID.String()})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

// --- UpdatePost ---

func TestUpdatePost_Success(t *testing.T) {
	content := "updated"
	post := newPost(testPostID, testUserID, &content)

	resource := &PostResourceMock{
		UpdateFunc: func(_ context.Context, id, userID uuid.UUID, mc *string) (*ent.Post, error) {
			if id != testPostID {
				t.Errorf("expected id %s, got %s", testPostID, id)
			}
			if userID != testUserID {
				t.Errorf("expected userID %s, got %s", testUserID, userID)
			}
			return post, nil
		},
	}
	publisher := &client.MessagePublisherMock{
		PublishFunc: func(topic string, _ ...*message.Message) error {
			if topic != "post.updated" {
				t.Errorf("expected topic post.updated, got %s", topic)
			}
			return nil
		},
	}
	svc := newTestService(resource, publisher)

	resp, err := svc.UpdatePost(authedCtx(testClaim()), &postv1.UpdatePostRequest{
		Id:              testPostID.String(),
		MarkdownContent: &content,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Post.Id != testPostID.String() {
		t.Errorf("expected post id %s, got %s", testPostID, resp.Post.Id)
	}
}

func TestUpdatePost_Unauthenticated(t *testing.T) {
	svc := newTestService(&PostResourceMock{}, &client.MessagePublisherMock{})

	_, err := svc.UpdatePost(context.Background(), &postv1.UpdatePostRequest{Id: testPostID.String()})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestUpdatePost_NotFound(t *testing.T) {
	resource := &PostResourceMock{
		UpdateFunc: func(_ context.Context, _, _ uuid.UUID, _ *string) (*ent.Post, error) {
			return nil, &ent.NotFoundError{}
		},
	}
	svc := newTestService(resource, &client.MessagePublisherMock{})

	_, err := svc.UpdatePost(authedCtx(testClaim()), &postv1.UpdatePostRequest{Id: testPostID.String()})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestUpdatePost_Forbidden(t *testing.T) {
	resource := &PostResourceMock{
		UpdateFunc: func(_ context.Context, _, _ uuid.UUID, _ *string) (*ent.Post, error) {
			return nil, ucpost.ErrForbidden
		},
	}
	svc := newTestService(resource, &client.MessagePublisherMock{})

	_, err := svc.UpdatePost(authedCtx(testClaim()), &postv1.UpdatePostRequest{Id: testPostID.String()})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}

// --- DeletePost ---

func TestDeletePost_Success(t *testing.T) {
	resource := &PostResourceMock{
		DeleteFunc: func(_ context.Context, id, userID uuid.UUID) error {
			if id != testPostID {
				t.Errorf("expected id %s, got %s", testPostID, id)
			}
			if userID != testUserID {
				t.Errorf("expected userID %s, got %s", testUserID, userID)
			}
			return nil
		},
	}
	publisher := &client.MessagePublisherMock{
		PublishFunc: func(topic string, msgs ...*message.Message) error {
			if topic != "post.deleted" {
				t.Errorf("expected topic post.deleted, got %s", topic)
			}
			if string(msgs[0].Payload) != testPostID.String() {
				t.Errorf("expected payload %s, got %s", testPostID, string(msgs[0].Payload))
			}
			return nil
		},
	}
	svc := newTestService(resource, publisher)

	_, err := svc.DeletePost(authedCtx(testClaim()), &postv1.DeletePostRequest{Id: testPostID.String()})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resource.DeleteCalls()) != 1 {
		t.Errorf("expected 1 Delete call, got %d", len(resource.DeleteCalls()))
	}
	if len(publisher.PublishCalls()) != 1 {
		t.Errorf("expected 1 Publish call, got %d", len(publisher.PublishCalls()))
	}
}

func TestDeletePost_Unauthenticated(t *testing.T) {
	svc := newTestService(&PostResourceMock{}, &client.MessagePublisherMock{})

	_, err := svc.DeletePost(context.Background(), &postv1.DeletePostRequest{Id: testPostID.String()})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestDeletePost_NotFound(t *testing.T) {
	resource := &PostResourceMock{
		DeleteFunc: func(_ context.Context, _, _ uuid.UUID) error {
			return &ent.NotFoundError{}
		},
	}
	svc := newTestService(resource, &client.MessagePublisherMock{})

	_, err := svc.DeletePost(authedCtx(testClaim()), &postv1.DeletePostRequest{Id: testPostID.String()})

	if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}
