package ent

import (
	"arktie.org/internal/data"
	postv1 "arktie.org/internal/pb/post/v1"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (p *Post) ToProto() *postv1.Post {
	pb := &postv1.Post{
		Id:        p.ID.String(),
		UserId:    p.UserID.String(),
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}

	if p.AtURL != nil {
		pb.AtUrl = *p.AtURL
	}
	if p.MarkdownContent != nil {
		pb.MarkdownContent = p.MarkdownContent
	}

	return pb
}

func (p *Post) ToPDSRecord() *data.PDSRecord {
	return &data.PDSRecord{
		Version:     "2026-04-05",
		Content:     p.MarkdownContent,
		CreatedAt:   syntax.Datetime(p.CreatedAt.Format(syntax.AtprotoDatetimeLayout)),
		UpdatedAt:   syntax.Datetime(p.UpdatedAt.Format(syntax.AtprotoDatetimeLayout)),
		PublishedAt: syntax.DatetimeNow(),
	}
}
