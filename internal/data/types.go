package data

import (
	"encoding/json"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type PDSRecord struct {
	Type          string          `mapstructure:"$type" json:"$type"`
	Version       string          `mapstructure:"version" json:"version"`
	Content       *string         `mapstructure:"content" json:"content"`
	CreatedAt     syntax.Datetime `mapstructure:"created_at" json:"created_at"`
	UpdatedAt     syntax.Datetime `mapstructure:"updated_at" json:"updated_at"`
	PublishedAt   syntax.Datetime `mapstructure:"published_at" json:"published_at"`
	PublishedFrom string          `mapstructure:"published_from" json:"published_from"`
}

func (r *PDSRecord) ToMap(from string) map[string]any {
	r.Type = "app.arktie.post"
	r.PublishedFrom = from

	data, err := json.Marshal(r)
	if err != nil {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
