package models

import "ingestion-service/db"

type Device struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TenantID    uint   `json:"tenant_id"`
	IsActive    bool   `json:"is_active"`
}

func AddDevice(device *Device) error {
	result := db.ORM.Create(device)
	return result.Error
}

func GetAllDevicesByTenantID(tenantID uint) ([]Device, error) {
	var devices []Device
	result := db.ORM.Where("tenant_id = ?", tenantID).Find(&devices)
	return devices, result.Error
}

func GetActiveDevicesByTenantID(tenantID uint) ([]Device, error) {
	var devices []Device
	result := db.ORM.Where("tenant_id = ? AND is_active = ?", tenantID, true).Find(&devices)
	return devices, result.Error
}

func GetDeviceByID(id uint) (*Device, error) {
	var device Device
	result := db.ORM.First(&device, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &device, nil
}

func UpdateDevice(id uint, updatedDevice *Device) error {
	var device Device
	result := db.ORM.First(&device, id)
	if result.Error != nil {
		return result.Error
	}
	device.Name = updatedDevice.Name
	device.Description = updatedDevice.Description
	device.TenantID = updatedDevice.TenantID
	device.IsActive = updatedDevice.IsActive
	return db.ORM.Save(&device).Error
}

func DeleteDevice(id uint) error {
	return db.ORM.Delete(&Device{}, id).Error
}