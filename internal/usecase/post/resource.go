package post

import (
	"context"

	"github.com/google/uuid"

	"arktie.org/ent"
	entpost "arktie.org/ent/post"
)

func (uc *Usecase) Create(ctx context.Context, userID uuid.UUID, markdownContent *string) (*ent.Post, error) {
	builder := uc.db.Ent.Post.Create().
		SetUserID(userID)

	if markdownContent != nil {
		builder.SetMarkdownContent(*markdownContent)
	}

	return builder.Save(ctx)
}

func (uc *Usecase) Get(ctx context.Context, id uuid.UUID) (*ent.Post, error) {
	return uc.db.Ent.Post.Get(ctx, id)
}

func (uc *Usecase) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, markdownContent *string) (*ent.Post, error) {
	p, err := uc.db.Ent.Post.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.UserID != userID {
		return nil, ErrForbidden
	}

	builder := uc.db.Ent.Post.UpdateOneID(id)
	if markdownContent != nil {
		builder.SetMarkdownContent(*markdownContent)
	}

	return builder.Save(ctx)
}

func (uc *Usecase) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	n, err := uc.db.Ent.Post.Delete().
		Where(entpost.ID(id), entpost.UserID(userID)).
		Exec(ctx)
	if err != nil {
		return err
	}

	if n == 0 {
		return &ent.NotFoundError{}
	}

	return nil
}
