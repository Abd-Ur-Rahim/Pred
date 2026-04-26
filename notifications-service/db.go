package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func initDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS device_tokens (
			id         BIGSERIAL   PRIMARY KEY,
			tenant_id  TEXT        NOT NULL,
			user_id    TEXT        NOT NULL,
			token      TEXT        NOT NULL,
			platform   TEXT        NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (token)
		);

		CREATE INDEX IF NOT EXISTS device_tokens_tenant_user
			ON device_tokens (tenant_id, user_id);

		CREATE TABLE IF NOT EXISTS notifications (
			id         BIGSERIAL   PRIMARY KEY,
			tenant_id  TEXT        NOT NULL,
			type       TEXT        NOT NULL,
			payload    JSONB       NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS notifications_tenant
			ON notifications (tenant_id);

		CREATE TABLE IF NOT EXISTS notification_deliveries (
			id              BIGSERIAL   PRIMARY KEY,
			notification_id BIGINT      NOT NULL REFERENCES notifications (id),
			tenant_id       TEXT        NOT NULL,
			user_id         TEXT        NOT NULL,
			recipient       TEXT        NOT NULL,
			device_token_id BIGINT      REFERENCES device_tokens (id),
			status          TEXT        NOT NULL DEFAULT 'pending',
			error           TEXT,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS notification_deliveries_notification
			ON notification_deliveries (notification_id);
		CREATE INDEX IF NOT EXISTS notification_deliveries_tenant_user
			ON notification_deliveries (tenant_id, user_id);
	`)
	return err
}

// --- device tokens ---

type DeviceToken struct {
	ID       int64
	UserID   string
	Token    string
	Platform string
}

func deviceTokensForUsers(ctx context.Context, pool *pgxpool.Pool, tenantID string, userIDs []string) ([]DeviceToken, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, token, platform
		FROM device_tokens
		WHERE tenant_id = $1 AND user_id = ANY($2)
	`, tenantID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []DeviceToken
	for rows.Next() {
		var t DeviceToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.Platform); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// --- notifications ---

func insertNotification(ctx context.Context, pool *pgxpool.Pool, tenantID, notifType string, payload []byte) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO notifications (tenant_id, type, payload)
		VALUES ($1, $2, $3)
		RETURNING id
	`, tenantID, notifType, payload).Scan(&id)
	return id, err
}

// --- notification_deliveries ---

func insertDelivery(ctx context.Context, pool *pgxpool.Pool, notificationID int64, tenantID, userID, recipient string, deviceTokenID *int64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO notification_deliveries (notification_id, tenant_id, user_id, recipient, device_token_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, notificationID, tenantID, userID, recipient, deviceTokenID).Scan(&id)
	return id, err
}

func updateDeliveryStatus(ctx context.Context, pool *pgxpool.Pool, id int64, status, errMsg string) error {
	var errVal *string
	if errMsg != "" {
		errVal = &errMsg
	}
	_, err := pool.Exec(ctx, `
		UPDATE notification_deliveries
		SET status = $1, error = $2, updated_at = NOW()
		WHERE id = $3
	`, status, errVal, id)
	return err
}
