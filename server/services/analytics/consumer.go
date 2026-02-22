package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/amnayem/skillforge/shared/pkg/logger"
	"github.com/segmentio/kafka-go"
)

type enrollmentEventPayload struct {
	UserID   string  `json:"user_id"`
	CourseID string  `json:"course_id"`
	TenantID string  `json:"tenant_id"`
	Progress float64 `json:"progress"`
	Event    string  `json:"event"`
}

// AnalyticsConsumer ingests Kafka events and indexes them to Elasticsearch.
type AnalyticsConsumer struct {
	reader *kafka.Reader
	repo   Repository
}

func NewAnalyticsConsumer(brokers []string, groupID string, repo Repository) *AnalyticsConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    "enrollment.created",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &AnalyticsConsumer{reader: r, repo: repo}
}

func (c *AnalyticsConsumer) Run(ctx context.Context) {
	log := logger.Get()
	log.Info().Msg("Analytics consumer started")
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info().Msg("Analytics consumer shutting down")
				return
			}
			log.Error().Err(err).Msg("Kafka read error")
			continue
		}
		c.index(ctx, msg)
	}
}

func (c *AnalyticsConsumer) Close() error {
	return c.reader.Close()
}

func (c *AnalyticsConsumer) index(ctx context.Context, msg kafka.Message) {
	log := logger.Get()
	var p enrollmentEventPayload
	if err := json.Unmarshal(msg.Value, &p); err != nil {
		log.Error().Err(err).Msg("analytics: unmarshal payload")
		return
	}
	p.Event = msg.Topic

	doc := map[string]interface{}{
		"user_id":    p.UserID,
		"course_id":  p.CourseID,
		"tenant_id":  p.TenantID,
		"progress":   p.Progress,
		"event":      p.Event,
		"@timestamp": fmt.Sprintf("%s", msg.Time.UTC().Format("2006-01-02T15:04:05Z")),
	}

	// Pretty-print for logging; actual index payload is the doc map.
	if prettyJSON, err := json.Marshal(doc); err == nil {
		log.Debug().RawJSON("doc", prettyJSON).Str("topic", msg.Topic).Msg("indexing event")
		_ = bytes.NewReader(prettyJSON)
	}

	if err := c.repo.IndexEvent(ctx, "analytics-enrollments", doc); err != nil {
		log.Error().Err(err).Msg("analytics: index event failed")
	}
}
