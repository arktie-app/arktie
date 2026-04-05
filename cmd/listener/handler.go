package main

import (
	"encoding/json"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"arktie.org/internal/server"
)

type FirehoseLogHandler struct{}

var _ server.FirehosePostHandler = &FirehoseLogHandler{}

func (h *FirehoseLogHandler) HandlePost(msg *message.Message) error {
	repo := msg.Metadata.Get("repo")
	path := msg.Metadata.Get("path")

	var payload any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		payload = string(msg.Payload)
	}

	jsonBytes, _ := json.MarshalIndent(payload, "", "  ")
	slog.Info("received firehose event",
		slog.String("repo", repo),
		slog.String("path", path),
		slog.String("payload", string(jsonBytes)),
	)

	return nil
}
