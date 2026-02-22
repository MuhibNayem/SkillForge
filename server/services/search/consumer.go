package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amnayem/skillforge/shared/pkg/logger"
	"github.com/segmentio/kafka-go"
)

type courseEventPayload struct {
	CourseID    string `json:"course_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
}

type contentUploadedPayload struct {
	ContentID   string `json:"content_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
}

// IndexConsumer listens for creation/update events and keeps the search index current.
type IndexConsumer struct {
	readers []*kafka.Reader
	repo    Repository
}

func NewIndexConsumer(brokers []string, groupID string, repo Repository) *IndexConsumer {
	topicConfigs := []kafka.ReaderConfig{
		{Brokers: brokers, GroupID: groupID + "-courses", Topic: "course.created", MinBytes: 1, MaxBytes: 10e6},
		{Brokers: brokers, GroupID: groupID + "-courses-upd", Topic: "course.updated", MinBytes: 1, MaxBytes: 10e6},
		{Brokers: brokers, GroupID: groupID + "-content", Topic: "content.uploaded", MinBytes: 1, MaxBytes: 10e6},
	}
	readers := make([]*kafka.Reader, len(topicConfigs))
	for i, cfg := range topicConfigs {
		readers[i] = kafka.NewReader(cfg)
	}
	return &IndexConsumer{readers: readers, repo: repo}
}

func (c *IndexConsumer) Run(ctx context.Context) {
	log := logger.Get()
	log.Info().Msg("Search index consumer started")
	// Fan-out: one goroutine per reader.
	done := make(chan struct{}, len(c.readers))
	for _, r := range c.readers {
		go func(reader *kafka.Reader) {
			defer func() { done <- struct{}{} }()
			for {
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Error().Err(err).Msg("search consumer: read error")
					continue
				}
				c.handleMessage(ctx, msg)
			}
		}(r)
	}
	// Wait for context cancellation.
	<-ctx.Done()
	log.Info().Msg("Search index consumer shutting down")
}

func (c *IndexConsumer) Close() {
	for _, r := range c.readers {
		_ = r.Close()
	}
}

func (c *IndexConsumer) handleMessage(ctx context.Context, msg kafka.Message) {
	log := logger.Get()
	switch msg.Topic {
	case "course.created", "course.updated":
		var p courseEventPayload
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Error().Err(err).Msg("search: unmarshal course event")
			return
		}
		doc := IndexDocument{
			ID:          p.CourseID,
			Type:        "course",
			Title:       p.Title,
			Description: p.Description,
			TenantID:    p.TenantID,
			URL:         fmt.Sprintf("/courses/%s", p.CourseID),
		}
		if err := c.repo.IndexDocument(ctx, doc); err != nil {
			log.Error().Err(err).Str("course_id", p.CourseID).Msg("search: index course failed")
		}

	case "content.uploaded":
		var p contentUploadedPayload
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Error().Err(err).Msg("search: unmarshal content event")
			return
		}
		doc := IndexDocument{
			ID:          p.ContentID,
			Type:        "content",
			Title:       p.Title,
			Description: p.Description,
			TenantID:    p.TenantID,
			URL:         fmt.Sprintf("/content/%s", p.ContentID),
		}
		if err := c.repo.IndexDocument(ctx, doc); err != nil {
			log.Error().Err(err).Str("content_id", p.ContentID).Msg("search: index content failed")
		}
	}
}
