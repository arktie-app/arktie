package server

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/internal/data/client"
	"arktie.org/internal/service/post"
)

// PostPusher publish posts which local created to PDS
type PostPusher interface {
	PushCreate(msg *message.Message) error
	PushUpdate(msg *message.Message) error
	PushDelete(msg *message.Message) error
}

var _ PostPusher = &post.SyncService{}

// PostPuller get posts from PDS and store into local
type PostPuller interface {
	PullCreate(msg *message.Message) error
	PullUpdate(msg *message.Message) error
	PullDelete(msg *message.Message) error
}

var _ PostPuller = &post.SyncService{}

type AppListener struct{}

func NewAppListener(
	event *client.Event,

	post PostPusher,
) *AppListener {
	event.Router.AddConsumerHandler(
		"post.created",
		"post.created",
		event.Subscriber,
		post.PushCreate,
	)

	event.Router.AddConsumerHandler(
		"post.updated",
		"post.updated",
		event.Subscriber,
		post.PushUpdate,
	)

	event.Router.AddConsumerHandler(
		"post.deleted",
		"post.deleted",
		event.Subscriber,
		post.PushDelete,
	)

	return &AppListener{}
}

type FirehoseListener struct{}

func NewFirehoseListener(
	event *client.Event,

	post PostPuller,
) *FirehoseListener {
	event.Router.AddConsumerHandler(
		"firehose.post.create",
		"firehose.post.create",
		event.Subscriber,
		post.PullCreate,
	)

	event.Router.AddConsumerHandler(
		"firehose.post.update",
		"firehose.post.update",
		event.Subscriber,
		post.PullUpdate,
	)

	event.Router.AddConsumerHandler(
		"firehose.post.delete",
		"firehose.post.delete",
		event.Subscriber,
		post.PullDelete,
	)

	return &FirehoseListener{}
}
