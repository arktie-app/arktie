package ent

import postv1 "arktie.org/internal/pb/post/v1"

func (p *Post) ToProto() *postv1.Post {
	pb := &postv1.Post{
		Id:        p.ID.String(),
		UserId:    p.UserID.String(),
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if p.AtURL != nil {
		pb.AtUrl = *p.AtURL
	}
	if p.MarkdownContent != nil {
		pb.MarkdownContent = p.MarkdownContent
	}

	return pb
}
