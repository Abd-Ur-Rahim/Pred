package handlers

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"strconv"
	"strings"
)

const (
	mqttTopicData         = "data"
	mqttTopicRegistration = "registration"
)

// parseDeviceTopic parses device MQTT topic in one place.
// Supported topics:
// - devices/{deviceID}/data
// - devices/{deviceID}/registration
func parseDeviceTopic(topic string) (uint, string, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" {
		return 0, "", fmt.Errorf("expected devices/{deviceID}/{kind}")
	}

	deviceID64, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, "", fmt.Errorf("invalid device id: %w", err)
	}

	switch parts[2] {
	case mqttTopicData, mqttTopicRegistration:
		return uint(deviceID64), parts[2], nil
	default:
		return 0, "", fmt.Errorf("unsupported topic kind: %s", parts[2])
	}
}

func verifySignature(pub *ecdsa.PublicKey, payload []byte, sig []byte) bool { // Placeholder for signature verification logic.
	hash := sha256.Sum256(payload)

	return ecdsa.VerifyASN1(pub, hash[:], sig)
}

func ParsePublicKey(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pub.(*ecdsa.PublicKey), nil
}

func verifyDeviceData(deviceID uint, fallbackPublicKey *string, payload []byte) error {
	var message struct {
		Nonce     string          `json:"nonce"`
		Payload   json.RawMessage `json:"payload"`
		Signature string          `json:"signature"`
	}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("invalid signed payload: %w", err)
	}
	if message.Nonce == "" {
		return fmt.Errorf("nonce is required")
	}
	if len(message.Payload) == 0 {
		return fmt.Errorf("payload field is required")
	}
	if message.Signature == "" {
		return fmt.Errorf("signature is required")
	}

	publicKeyPEM, err := resolveDevicePublicKey(deviceID, fallbackPublicKey)
	if err != nil {
		return err
	}

	publicKey, err := ParsePublicKey(publicKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	if redisCache != nil {
		exists, err := redisCache.NonceExists(context.Background(), deviceID, message.Nonce)
		if err != nil {
			return fmt.Errorf("nonce check failed: %w", err)
		}
		if exists {
			return fmt.Errorf("replayed nonce")
		}
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(message.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	if !verifySignature(publicKey, message.Payload, signatureBytes) {
		return fmt.Errorf("signature verification failed")
	}

	if redisCache != nil {
		marked, err := redisCache.MarkNonceUsed(context.Background(), deviceID, message.Nonce)
		if err != nil {
			return fmt.Errorf("failed to mark nonce used: %w", err)
		}
		if !marked {
			return fmt.Errorf("replayed nonce")
		}
	}

	log.Printf("verified signed payload for device_id=%d payload_bytes=%d", deviceID, len(message.Payload))
	return nil
}

func resolveDevicePublicKey(deviceID uint, fallbackPublicKey *string) (string, error) {
	if redisCache != nil {
		cachedKey, err := redisCache.GetDevicePublicKey(context.Background(), deviceID)
		if err == nil && cachedKey != "" {
			return cachedKey, nil
		}
	}

	if fallbackPublicKey == nil || *fallbackPublicKey == "" {
		return "", fmt.Errorf("device public key not found")
	}

	key := *fallbackPublicKey
	if redisCache != nil {
		if err := redisCache.CacheDevicePublicKey(context.Background(), deviceID, key); err != nil {
			log.Printf("failed to cache public key for device_id=%d: %v", deviceID, err)
		}
	}

	return key, nil
}
