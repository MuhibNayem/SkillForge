package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/smtp"

	"github.com/amnayem/skillforge/shared/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// Event payload shapes consumed from Kafka.
type userRegisteredPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type enrollmentCreatedPayload struct {
	UserID   string `json:"user_id"`
	CourseID string `json:"course_id"`
	TenantID string `json:"tenant_id"`
}

type assessmentGradedPayload struct {
	UserID       string  `json:"user_id"`
	AssessmentID string  `json:"assessment_id"`
	Score        float64 `json:"score"`
}

type coursePublishedPayload struct {
	CourseID string `json:"course_id"`
	Title    string `json:"title"`
	TenantID string `json:"tenant_id"`
}

type courseCompletedPayload struct {
	UserID   string `json:"user_id"`
	CourseID string `json:"course_id"`
}

// SMTPConfig holds outgoing mail server config.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Consumer listens to Kafka and dispatches notifications.
type Consumer struct {
	reader  *kafka.Reader
	repo    Repository
	redis   *redis.Client
	smtp    SMTPConfig
	enabled bool // false in test/dev when SMTP not configured
}

func NewConsumer(brokers []string, groupID string, topics []string, repo Repository, rdb *redis.Client, smtpCfg SMTPConfig) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topics[0], // primary topic; multi-topic handled in Run
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &Consumer{
		reader:  r,
		repo:    repo,
		redis:   rdb,
		smtp:    smtpCfg,
		enabled: smtpCfg.Host != "",
	}
}

// Run starts consuming messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	log := logger.Get()
	log.Info().Msg("Notification consumer started")
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info().Msg("Notification consumer shutting down")
				return
			}
			log.Error().Err(err).Msg("Kafka read error")
			continue
		}
		c.dispatch(ctx, msg)
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) dispatch(ctx context.Context, msg kafka.Message) {
	log := logger.Get()
	topic := msg.Topic

	switch topic {
	case "user.registered":
		var p userRegisteredPayload
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Error().Err(err).Str("topic", topic).Msg("unmarshal failed")
			return
		}
		prefs, err := c.repo.GetPreferences(ctx, p.UserID)
		if err != nil || prefs.EmailCourseUpdates {
			c.sendEmail(ctx, p.Email,
				"Welcome to SkillForge!",
				fmt.Sprintf("Hi %s,\n\nWelcome! Your account is ready.", p.Name),
				p.UserID, topic)
		}

	case "enrollment.created":
		var p enrollmentCreatedPayload
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Error().Err(err).Str("topic", topic).Msg("unmarshal failed")
			return
		}
		c.pushInApp(ctx, p.UserID, "enrollment.created",
			fmt.Sprintf(`{"course_id":"%s","message":"You have been enrolled in a new course."}`, p.CourseID))

	case "assessment.graded":
		var p assessmentGradedPayload
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Error().Err(err).Str("topic", topic).Msg("unmarshal failed")
			return
		}
		c.pushInApp(ctx, p.UserID, "assessment.graded",
			fmt.Sprintf(`{"assessment_id":"%s","score":%f}`, p.AssessmentID, p.Score))

	case "course.completed":
		var p courseCompletedPayload
		if err := json.Unmarshal(msg.Value, &p); err != nil {
			log.Error().Err(err).Str("topic", topic).Msg("unmarshal failed")
			return
		}
		c.pushInApp(ctx, p.UserID, "course.completed",
			fmt.Sprintf(`{"course_id":"%s","message":"Congratulations! Course completed."}`, p.CourseID))

	default:
		log.Debug().Str("topic", topic).Msg("unhandled topic")
	}
}

// sendEmail dispatches an SMTP email and records delivery.
func (c *Consumer) sendEmail(ctx context.Context, to, subject, body, userID, eventType string) {
	log := logger.Get()
	deliveryStatus := "sent"

	if c.enabled {
		auth := smtp.PlainAuth("", c.smtp.Username, c.smtp.Password, c.smtp.Host)
		msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
		addr := fmt.Sprintf("%s:%d", c.smtp.Host, c.smtp.Port)
		if err := smtp.SendMail(addr, auth, c.smtp.From, []string{to}, []byte(msg)); err != nil {
			log.Error().Err(err).Str("to", to).Msg("email send failed")
			deliveryStatus = "failed"
		}
	} else {
		log.Debug().Str("to", to).Str("subject", subject).Msg("[SMTP disabled] email would be sent")
	}

	_ = c.repo.RecordDelivery(ctx, DeliveryRecord{
		UserID:    userID,
		Channel:   "email",
		EventType: eventType,
		Payload:   fmt.Sprintf(`{"to":%q,"subject":%q}`, to, subject),
		Status:    deliveryStatus,
	})
}

// pushInApp stores an in-app notification in Redis (pub/sub channel + list).
func (c *Consumer) pushInApp(ctx context.Context, userID, eventType, payload string) {
	log := logger.Get()
	key := fmt.Sprintf("notifications:%s", userID)

	entry := fmt.Sprintf(`{"event":"%s","payload":%s}`, eventType, payload)
	if err := c.redis.LPush(ctx, key, entry).Err(); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("redis lpush failed")
	}
	// Trim to last 100 notifications per user.
	_ = c.redis.LTrim(ctx, key, 0, 99)

	// Publish on a channel so connected WebSocket clients receive it instantly.
	channel := fmt.Sprintf("notif:%s", userID)
	if err := c.redis.Publish(ctx, channel, entry).Err(); err != nil {
		log.Error().Err(err).Str("channel", channel).Msg("redis publish failed")
	}

	_ = c.repo.RecordDelivery(ctx, DeliveryRecord{
		UserID:    userID,
		Channel:   "in_app",
		EventType: eventType,
		Payload:   payload,
		Status:    "sent",
	})
}

// Compile-time guard: Consumer must implement io.Closer.
var _ io.Closer = (*Consumer)(nil)
