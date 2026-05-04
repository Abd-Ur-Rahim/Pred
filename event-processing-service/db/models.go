package db

import (
	"time"

	"gorm.io/datatypes"
)

type Event struct {
	ID        int64          `gorm:"primaryKey"`
	TenantID  string         `gorm:"not null;index:events_tenant"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
}
