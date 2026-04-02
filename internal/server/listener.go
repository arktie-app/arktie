package server

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/internal/data"
	"arktie.org/internal/service/post"
)

type Listener struct{}

func NewListener(
	client *data.Client,

	postSyncher PostSyncher,
) *Listener {
	client.Router.AddConsumerHandler(
		"post.created",
		"post.created",
		client.Subscriber,
		postSyncher.Create,
	)

	return &Listener{}
}

type PostSyncher interface {
	Create(msg *message.Message) error
	Update(msg *message.Message) error
	Delete(msg *message.Message) error
}

var _ PostSyncher = &post.Service{}
