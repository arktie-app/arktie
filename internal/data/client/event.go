package client

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"arktie.org/internal/data"
)

type Event struct {
	Publisher  message.Publisher
	Subscriber message.Subscriber
	Router     *message.Router
}

func NewEvent(cfg *data.Config) (client *Event, err error) {
	client = &Event{}

	if err = client.initWatermill(cfg); err != nil {
		return
	}

	return
}

func (c *Event) initWatermill(cfg *data.Config) error {
	logger := watermill.NewStdLogger(cfg.App.Debug, cfg.App.Debug)

	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	c.Publisher = pubSub
	c.Subscriber = pubSub

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return err
	}

	// Router level middleware are executed for every message sent to the router
	router.AddMiddleware(
		// CorrelationID will copy the correlation id from the incoming message's metadata to the produced messages
		middleware.CorrelationID,

		// The handler function is retried if it returns an error.
		// After MaxRetries, the message is Nacked and it's up to the PubSub to resend it.
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: time.Millisecond * 100,
			Logger:          logger,
		}.Middleware,

		// Recoverer handles panics from handlers.
		// In this case, it passes them as errors to the Retry middleware.
		middleware.Recoverer,
	)

	c.Router = router
	return nil
}

//go:generate go tool moq -rm -out mock_message.go . MessagePublisher MessageSubscriber

// MessagePublisher wraps watermill's message.Publisher for mock generation.
type MessagePublisher = message.Publisher

// MessageSubscriber wraps watermill's message.Subscriber for mock generation.
type MessageSubscriber = message.Subscriber
