package server

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/internal/data/client"
	"arktie.org/internal/service/post"
)

type Listener struct{}

func NewListener(
	event *client.Event,

	postSyncher PostSyncher,
) *Listener {
	event.Router.AddConsumerHandler(
		"post.created",
		"post.created",
		event.Subscriber,
		postSyncher.Create,
	)

	event.Router.AddConsumerHandler(
		"post.updated",
		"post.updated",
		event.Subscriber,
		postSyncher.Update,
	)

	event.Router.AddConsumerHandler(
		"post.deleted",
		"post.deleted",
		event.Subscriber,
		postSyncher.Delete,
	)

	return &Listener{}
}

type PostSyncher interface {
	Create(msg *message.Message) error
	Update(msg *message.Message) error
	Delete(msg *message.Message) error
}

var _ PostSyncher = &post.Service{}
