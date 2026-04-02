package ent

import (
	postv1 "arktie.org/internal/pb/post/v1"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

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

func (p *Post) ToPDSRecord() map[string]any {
	return map[string]any{
		"$type":        "app.arktie.post",
		"version":      "2026-04-03",
		"content":      p.MarkdownContent,
		"created_at":   syntax.Datetime(p.CreatedAt.Format(syntax.AtprotoDatetimeLayout)),
		"updated_at":   syntax.Datetime(p.UpdatedAt.Format(syntax.AtprotoDatetimeLayout)),
		"published_at": syntax.DatetimeNow(),
	}
}
