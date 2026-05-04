package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"notifications-service/db"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping consumer integration tests")
	}
	ctx := context.Background()
	gdb, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return gdb
}

func makeMessage(t *testing.T, v any) kafka.Message {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	return kafka.Message{Value: b}
}

func TestHandleMessage_Email(t *testing.T) {
	gdb := openTestDB(t)
	ctx := context.Background()

	event := AlertEvent{
		TenantID: "t-email",
		Type:     "email",
		Payload:  json.RawMessage(`{"subject":"Test","body":"Hello"}`),
		Recipients: []Recipient{
			{UserID: "u1", Email: "u1@example.com"},
			{UserID: "u2", Email: "u2@example.com"},
		},
	}

	if err := handleMessage(ctx, gdb, makeMessage(t, event)); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}

	var notif db.Notification
	if err := gdb.Where("tenant_id = ? AND type = ?", "t-email", "email").Last(&notif).Error; err != nil {
		t.Fatalf("fetch notification: %v", err)
	}

	var deliveries []db.NotificationDelivery
	gdb.Where("notification_id = ?", notif.ID).Find(&deliveries)
	if len(deliveries) != 2 {
		t.Fatalf("delivery count: got %d, want 2", len(deliveries))
	}
	for _, d := range deliveries {
		if d.Status != "delivered" {
			t.Errorf("delivery %d status: got %q, want %q", d.ID, d.Status, "delivered")
		}
	}
}

func TestHandleMessage_Push(t *testing.T) {
	gdb := openTestDB(t)
	ctx := context.Background()

	// Seed device tokens.
	tokens := []db.DeviceToken{
		{TenantID: "t-push", UserID: "u10", Token: "push-tok-u10", Platform: "ios"},
		{TenantID: "t-push", UserID: "u11", Token: "push-tok-u11", Platform: "android"},
	}
	for i := range tokens {
		if err := gdb.Create(&tokens[i]).Error; err != nil {
			t.Fatalf("seed token: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, tok := range tokens {
			gdb.Delete(&db.DeviceToken{}, tok.ID)
		}
	})

	event := AlertEvent{
		TenantID: "t-push",
		Type:     "push",
		Payload:  json.RawMessage(`{"title":"Alert","body":"Check this out"}`),
		Recipients: []Recipient{
			{UserID: "u10"},
			{UserID: "u11"},
		},
	}

	if err := handleMessage(ctx, gdb, makeMessage(t, event)); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}

	var notif db.Notification
	if err := gdb.Where("tenant_id = ? AND type = ?", "t-push", "push").Last(&notif).Error; err != nil {
		t.Fatalf("fetch notification: %v", err)
	}

	var deliveries []db.NotificationDelivery
	gdb.Where("notification_id = ?", notif.ID).Find(&deliveries)
	if len(deliveries) != 2 {
		t.Fatalf("delivery count: got %d, want 2", len(deliveries))
	}
	for _, d := range deliveries {
		if d.Status != "delivered" {
			t.Errorf("delivery %d status: got %q, want %q", d.ID, d.Status, "delivered")
		}
		if d.DeviceTokenID == nil {
			t.Errorf("delivery %d: expected DeviceTokenID to be set", d.ID)
		}
	}
}

func TestHandleMessage_UnknownType(t *testing.T) {
	gdb := openTestDB(t)
	ctx := context.Background()

	event := AlertEvent{
		TenantID:   "t-unknown",
		Type:       "sms",
		Payload:    json.RawMessage(`{}`),
		Recipients: []Recipient{{UserID: "u1"}},
	}

	err := handleMessage(ctx, gdb, makeMessage(t, event))
	if err == nil {
		t.Fatal("expected error for unknown notification type, got nil")
	}
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	gdb := openTestDB(t)
	ctx := context.Background()

	msg := kafka.Message{Value: []byte(`not valid json`)}
	err := handleMessage(ctx, gdb, msg)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
