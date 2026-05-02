# Ingestion-Service Architecture

This document describes the architecture of the `ingestion-service` (device-facing entrypoint), its responsibilities, inputs/outputs, deployment surfaces, and integration considerations. It is scoped to the ingestion microservice only.

## Purpose and responsibilities
- Accept telemetry from devices via MQTT and HTTP.
- Normalize and validate telemetry payloads.
- Enrich messages with metadata (tenant_id, received_at, provenance).
- Publish normalized events to Kafka (`events` topic) for downstream services to consume.

## Component overview
- MQTT broker client: subscribes/accepts device topics (e.g., `devices/{tenant_id}/{device_id}/telemetry`) and translates MQTT payloads to normalized events.
- HTTP ingest endpoint: lightweight REST endpoint for devices/gateways to POST telemetry.
- Normalizer & validator: enforces schema (required fields, timestamps, types) and drops or NACKs invalid payloads.
- Kafka producer: publishes canonical event JSON to the configured `KAFKA_TOPIC_EVENTS`.
- Health + metrics endpoint: exposes `/health` and `/metrics` for orchestration and monitoring.

## Logical flow
1. Device publishes telemetry via MQTT or POSTs JSON to HTTP API.
2. Ingest receives raw payload and extracts `tenant_id` (required). If missing, reject with HTTP 400 or drop + log for MQTT clients.
3. Normalizer validates fields and adds `received_at` in UTC and a unique `message_id` if absent.
4. Producer publishes canonical event to Kafka (`events` topic) with key set to `tenant_id` or `device_id` (partitioning strategy).
5. Service returns success ack to device/gateway (HTTP 200 / MQTT PUBACK).

## MQTT Device Data Payload (device → ingestion)

Devices publish signed telemetry to `devices/{deviceID}/data`. The payload envelope must contain an ECDSA signature over the exact bytes of the `data` object.

```json
{
  "timestamp": 1704067200,
  "nonce": "unique-nonce-per-message",
  "data": {
    "mode": "normal",
    "peak_hz_1": 50,
    "peak_hz_2": 100,
    "peak_hz_3": 150,
    "status": "ok",
    "temp_c": 72.4,
    "v_rms": 1.23
  },
  "signature": "BASE64_ENCODED_ECDSA_SHA256_SIGNATURE"
}
```

**Signature Verification Logic:**
1. Extract `data` as raw bytes from the JSON payload (not re-marshaled).
2. Compute SHA256 hash of those bytes.
3. Verify ECDSA signature against the hash using the device's registered public key.
4. Check `nonce` for replay attacks (Redis-backed list of used nonces per device).
5. If all checks pass, unmarshal `data` and forward to Kafka.

**Important**: JSON field order in `data` must be deterministic (canonically ordered). If device and server marshal JSON differently, signature verification will fail.

## Kafka Output Payload (ingestion → Kafka)

The ingestion service publishes the sensor data to Kafka with device metadata:

```json
{
  "device_id": 1,
  "timestamp": 1704067200,
  "mode": "normal",
  "v_rms": 1.23,
  "temp_c": 72.4,
  "peak_hz_1": 50,
  "peak_hz_2": 100,
  "peak_hz_3": 150,
  "status": "ok"
}
```

- `message_id` should be a stable id for idempotency/deduplication downstream.
- `timestamp` is device-supplied event time; `received_at` is ingestion time.

## Topics, keys and partitioning
- Topic: `events` (configurable via `KAFKA_TOPIC_EVENTS`).
- Producer key: prefer `tenant_id` for co-location by tenant, or `device_id` for strict per-device ordering. Document your choice in deployment config.

## Environment and configuration (key vars)
- `KAFKA_BROKERS` — e.g., `localhost:9092`
- `KAFKA_TOPIC_EVENTS` — default `events`
- `KAFKA_GROUP_ID` — used when ingestion contains any consumer parts (optional)
- `MQTT_BROKER_URL` — URL for MQTT broker (e.g., `tcp://mosquitto:1883`)
- `HTTP_BIND_ADDR` — HTTP listen address (e.g., `:8080`)
- `LOG_LEVEL` — logging verbosity
- `DATABASE_URL` — only if the ingestion service needs a local DB for dedupe/offsets (not required in current implementation)

## Security
- Require `tenant_id` in every message. If you support multi-tenant devices, authenticate devices and map credentials to tenant.
- For production: enable TLS on MQTT and HTTPS for HTTP endpoints; configure client certs or token-based auth.
- Secure Kafka with TLS and SASL in production.

## Observability & health
- Expose `/health` for liveness and readiness.
- Expose `/metrics` for Prometheus scraping (request rate, success/failure counts, Kafka publish latency, etc.).
- Log structured JSON including `tenant_id`, `device_id`, `message_id` for traceability.

## Operational considerations
- Backpressure: if Kafka is unavailable, the service should buffer to local disk (or return 503 for HTTP). Avoid unbounded memory buffering.
- Retries: implement limited retries with exponential backoff for Kafka publish failures; consider a dead-letter topic for messages failing schema validation or persistent failures.
- Idempotency: downstream consumers will use `message_id` for deduplication. Ensure the ingestion service generates stable IDs if retries occur.

## Testing & local run
- Use the repo `docker-compose.yml` to bring up a local MQTT broker (mosquitto) and Kafka/Postgres as needed.
- Quick test commands (examples):

```sh
# publish MQTT test message
mosquitto_pub -h localhost -p 1883 -t "devices/tenant-123/device-001/telemetry" -m '{"device_id":"device-001","tenant_id":"tenant-123","timestamp":"2026-05-02T15:04:05Z","metrics":{"temperature_c":72.5}}'

# POST to HTTP ingest
curl -X POST http://localhost:8080/ingest -H 'Content-Type: application/json' -d '{"tenant_id":"tenant-123","device_id":"device-001","timestamp":"2026-05-02T15:04:05Z","metrics":{"temperature_c":72.5}}'
```

## Integrations (downstream)
- `event-processing-service` consumes `events` topic — coordinate the `KAFKA_TOPIC_EVENTS` name and partitioning key.
- Alerting/notification flows depend on downstream processors; ingestion should not emit alerts directly.

---
For step-by-step integration commands and payload examples, see the ingestion-focused integration guide: INTEGRATION.md.
