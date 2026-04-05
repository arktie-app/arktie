package server

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/internal/data/client"
	"arktie.org/internal/service/post"
)

type AppListener struct{}

func NewListener(
	event *client.Event,

	postSyncher PostSyncher,
) *AppListener {
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

	return &AppListener{}
}

type PostSyncher interface {
	Create(msg *message.Message) error
	Update(msg *message.Message) error
	Delete(msg *message.Message) error
}

var _ PostSyncher = &post.Service{}

type FirehoseListener struct{}

func NewFirehoseListener(
	event *client.Event,

	handler FirehosePostHandler,
) *FirehoseListener {
	event.Router.AddConsumerHandler(
		"firehose.post.create",
		"firehose.post.create",
		event.Subscriber,
		handler.HandlePost,
	)

	event.Router.AddConsumerHandler(
		"firehose.post.update",
		"firehose.post.update",
		event.Subscriber,
		handler.HandlePost,
	)

	event.Router.AddConsumerHandler(
		"firehose.post.delete",
		"firehose.post.delete",
		event.Subscriber,
		handler.HandlePost,
	)

	return &FirehoseListener{}
}

type FirehosePostHandler interface {
	HandlePost(msg *message.Message) error
}
