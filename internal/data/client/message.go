package client

import "github.com/ThreeDotsLabs/watermill/message"

//go:generate go tool moq -rm -out mock_message.go . MessagePublisher MessageSubscriber

// MessagePublisher wraps watermill's message.Publisher for mock generation.
type MessagePublisher = message.Publisher

// MessageSubscriber wraps watermill's message.Subscriber for mock generation.
type MessageSubscriber = message.Subscriber
