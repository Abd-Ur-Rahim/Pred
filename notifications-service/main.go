package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
	"github.com/joho/godotenv"

	"notifications-service/db"
)

// AlertEvent is the message schema consumed from Kafka.
// For push, Recipients carry user_ids resolved to device tokens via the device_tokens table.
// For email, Recipients carry user_ids and the email address to deliver to.
type AlertEvent struct {
	TenantID   string          `json:"tenant_id"`
	Type       string          `json:"type"` // "email" or "push"
	Payload    json.RawMessage `json:"payload"`
	Recipients []Recipient     `json:"recipients"`
}

type Recipient struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"` // required for type="email"
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	topic := getEnv("KAFKA_TOPIC", "notifications")
	groupID := getEnv("KAFKA_GROUP_ID", "notifications-service")
	dbURL := getEnv("DATABASE_URL", "postgres://localhost:5432/notifications")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gdb, err := db.Open(ctx, dbURL)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if sqlDB, err := gdb.DB(); err == nil {
		defer sqlDB.Close()
	}
	log.Println("connected to database")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(brokers, ","),
		Topic:   topic,
		GroupID: groupID,
	})
	defer reader.Close()

	log.Printf("consuming topic %q from %s (group %q)", topic, brokers, groupID)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("shutting down")
				return
			}
			log.Printf("read error: %v", err)
			continue
		}

		if err := handleMessage(ctx, gdb, msg); err != nil {
			log.Printf("handle error (offset %d): %v", msg.Offset, err)
		}
	}
}

func handleMessage(ctx context.Context, gdb *gorm.DB, msg kafka.Message) error {
	var event AlertEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	notifID, err := db.InsertNotification(ctx, gdb, event.TenantID, event.Type, event.Payload)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	switch event.Type {
	case "push":
		return fanOutPush(ctx, gdb, notifID, event)
	case "email":
		return fanOutEmail(ctx, gdb, notifID, event)
	default:
		return fmt.Errorf("unknown notification type %q", event.Type)
	}
}

func fanOutPush(ctx context.Context, gdb *gorm.DB, notifID int64, event AlertEvent) error {
	userIDs := make([]string, len(event.Recipients))
	for i, r := range event.Recipients {
		userIDs[i] = r.UserID
	}

	tokens, err := db.DeviceTokensForUsers(ctx, gdb, event.TenantID, userIDs)
	if err != nil {
		return fmt.Errorf("lookup device tokens: %w", err)
	}

	for _, t := range tokens {
		deliveryID, err := db.InsertDelivery(ctx, gdb, notifID, event.TenantID, t.UserID, t.Token, &t.ID)
		if err != nil {
			log.Printf("insert delivery failed (user %s, token %s): %v", t.UserID, t.Token, err)
			continue
		}

		status, errMsg := "delivered", ""
		if pushErr := sendPush(t.Token, t.Platform, event.Payload); pushErr != nil {
			log.Printf("push failed (user %s, token %s): %v", t.UserID, t.Token, pushErr)
			status, errMsg = "failed", pushErr.Error()
		}

		if err := db.UpdateDeliveryStatus(ctx, gdb, deliveryID, status, errMsg); err != nil {
			log.Printf("update status failed (delivery %d): %v", deliveryID, err)
		}
	}
	return nil
}

func fanOutEmail(ctx context.Context, gdb *gorm.DB, notifID int64, event AlertEvent) error {
	for _, r := range event.Recipients {
		deliveryID, err := db.InsertDelivery(ctx, gdb, notifID, event.TenantID, r.UserID, r.Email, nil)
		if err != nil {
			log.Printf("insert delivery failed (user %s, email %s): %v", r.UserID, r.Email, err)
			continue
		}

		status, errMsg := "delivered", ""
		if emailErr := sendEmail(r.Email, event.Payload); emailErr != nil {
			log.Printf("email failed (user %s, email %s): %v", r.UserID, r.Email, emailErr)
			status, errMsg = "failed", emailErr.Error()
		}

		if err := db.UpdateDeliveryStatus(ctx, gdb, deliveryID, status, errMsg); err != nil {
			log.Printf("update status failed (delivery %d): %v", deliveryID, err)
		}
	}
	return nil
}

func sendPush(token, platform string, payload json.RawMessage) error {
	log.Printf("push → %s (%s)", token, platform)
	return nil
}

func sendEmail(email string, payload json.RawMessage) error {
	log.Printf("email → %s", email)
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
