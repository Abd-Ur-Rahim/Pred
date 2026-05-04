package db

import (
	"context"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func InsertEvent(ctx context.Context, gdb *gorm.DB, tenantID string, payload []byte) (int64, error) {
	e := Event{TenantID: tenantID, Payload: datatypes.JSON(payload)}
	if err := gdb.WithContext(ctx).Create(&e).Error; err != nil {
		return 0, err
	}
	return e.ID, nil
}
