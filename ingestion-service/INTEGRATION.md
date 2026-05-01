# Ingestion-Service Integration

This guide covers only the `ingestion-service` integration: how to send telemetry (MQTT/HTTP), the canonical payloads produced to Kafka, environment variables needed, and test commands.

## Quick dev setup
1. Start local infra (Kafka, optional mosquitto) if not already running:

```sh
docker-compose up -d
```

2. Run the `ingestion-service` locally:

```sh
cd ingestion-service
go run .
```

3. Confirm service health:

```sh
curl http://localhost:8080/health
```

## Required environment variables (minimum)
- `KAFKA_BROKERS` (e.g. `localhost:9092`)
- `KAFKA_TOPIC_EVENTS` (default: `events`)
- `MQTT_BROKER_URL` (if using MQTT integration, e.g. `tcp://localhost:1883`)
- `HTTP_BIND_ADDR` (e.g. `:8080`)
- `LOG_LEVEL`

Place these in `ingestion-service/.env` or export them before running.

## Supported ingestion methods
- MQTT: devices publish to topics structured as `devices/{tenant_id}/{device_id}/telemetry`.
- HTTP: POST JSON to `/ingest` (or configured path). The request must contain `tenant_id` and `device_id`.

## Canonical ingestion payload (device -> ingestion)
Device-sent example (raw) — minimal required fields:

```json
{
  "tenant_id": "tenant-123",
  "device_id": "device-001",
  "timestamp": "2026-05-02T15:04:00Z",
  "metrics": {"temperature_c": 72.5}
}
```

The ingestion service will validate and transform it into the canonical Kafka event (see next section).

## Canonical Kafka event (ingestion -> Kafka `events`)
Always include `tenant_id` and a stable `message_id` for dedupe:

```json
{
  "message_id": "uuid-...",
  "tenant_id": "tenant-123",
  "device_id": "device-001",
  "received_at": "2026-05-02T15:04:05Z",
  "timestamp": "2026-05-02T15:04:00Z",
  "metrics": {"temperature_c": 72.5},
  "meta": {"source":"http","firmware_version":"1.2.3"}
}
```

Producer note: set the Kafka message key to `tenant_id` (or `device_id` if per-device ordering is required).

## Test commands (examples)

- Publish via MQTT (using `mosquitto_pub`):

```sh
mosquitto_pub -h localhost -p 1883 -t "devices/tenant-123/device-001/telemetry" -m '{"tenant_id":"tenant-123","device_id":"device-001","timestamp":"2026-05-02T15:04:00Z","metrics":{"temperature_c":72.5}}'
```

- POST via HTTP:

```sh
curl -X POST http://localhost:8080/ingest \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"tenant-123","device_id":"device-001","timestamp":"2026-05-02T15:04:00Z","metrics":{"temperature_c":72.5}}'
```

- Produce directly to Kafka (for downstream e2e tests):

```sh
kafka-console-producer --broker-list localhost:9092 --topic events
{"message_id":"test-1","tenant_id":"tenant-123","device_id":"device-001","received_at":"2026-05-02T15:04:05Z","timestamp":"2026-05-02T15:04:00Z","metrics":{"temperature_c":72.5}}
```

## Validation and verification
- After sending a message, consume the `events` topic to verify the canonical event appears:

```sh
kafka-console-consumer --bootstrap-server localhost:9092 --topic events --from-beginning
```

- Check `ingestion-service` logs for structured entries with `message_id` and `tenant_id`.

## Integration considerations (concise)
- Tenant propagation: `tenant_id` is mandatory and must match downstream tenant mapping.
- Idempotency: generate or forward `message_id` to allow downstream dedup.
- Ordering: if needed, partition by `device_id`.
- Error handling: invalid payloads should be rejected with a clear error for HTTP or logged and sent to a dead-letter topic for MQTT.

## Security notes
- For production, require TLS for HTTP and MQTT and authenticate devices (tokens or client certs).
- Secure Kafka with TLS and SASL; do not expose brokers directly to the public internet.

## Troubleshooting
- If messages are not appearing on `events`, check:
  - `ingestion-service` logs for publish errors.
  - Kafka broker connectivity (`KAFKA_BROKERS`).
  - Topic name (`KAFKA_TOPIC_EVENTS`) mismatch.

---
If you'd like, I can add a small sample test harness that publishes MQTT and HTTP messages and verifies Kafka consumption. Would you like that? 
## Database names used in tests / dev
