# Kong API Gateway

This directory contains the declarative configuration for Kong, which acts as the single entry point (API Gateway) for all frontend API requests.

## Architecture

Kong acts as the single entry point for all frontend API requests, routing traffic securely to the internal microservices running on the Docker network.

- **Proxy Port**: `8000` (Main API endpoint for frontend)
- **Admin Port**: `8002` (Kong Admin API, mapped from internal 8001)

### Services & Routes

Kong uses `strip_path: true` to remove the prefix before forwarding to the upstream service:

| Kong Path | Upstream Service | Example |
|-----------|-----------------|---------|
| `/api/ingest/*` | `ingestion-service:8003` | `GET /api/ingest/devices` → `GET /devices` |
| `/api/events/*` | `event-processing-service:8001` | `GET /api/events/health` → `GET /health` |

### DB-less Mode
Kong is configured in **DB-less mode**. Instead of relying on a Postgres database, its entire routing and plugin configuration is loaded directly into memory from the declarative `kong.yml` file. This makes the gateway lightweight and easy to version control.

### Plugins Enabled
- **CORS**: Handles Cross-Origin Resource Sharing so the web frontend can safely communicate with the API Gateway.
- **Rate Limiting**: Restricts API calls to 60 requests per minute per IP to prevent abuse and accidental spikes.
- **Request Size Limiting**: Prevents payloads larger than 10MB to protect the ingestion service from DoS attacks.

## Verification Guide

### 1. Check Kong Admin API
```bash
curl http://localhost:8002/
```
You should see a large JSON payload describing the Kong node configuration.

### 2. Verify Routing
Test the ingestion service through Kong:
```bash
curl -i http://localhost:8000/api/ingest/health
```
Expected: `200 OK` with `{"status":"ok"}`

Test the event-processing service through Kong:
```bash
curl -i http://localhost:8000/api/events/health
```
Expected: `200 OK` with `{"status":"ok"}`

Test a business endpoint:
```bash
curl -i http://localhost:8000/api/ingest/devices
```
Expected: `400 Bad Request` with `{"error":"tenant_id query param is required"}` (proves routing works end-to-end).

### 3. Verify Rate Limiting
Send 65 rapid requests to the gateway:
```bash
for i in {1..65}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8000/api/ingest/health; done
```
You should see `200` responses initially, and eventually `429 Too Many Requests` once the 60-request quota is exceeded.

### 4. Verify CORS Headers
```bash
curl -i -X OPTIONS http://localhost:8000/api/ingest/health \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET"
```
You should see `Access-Control-Allow-Origin: *` in the response headers.

## Troubleshooting

### "failure to get a peer from the ring-balancer"
This means Kong's active health checks have marked the upstream targets as unhealthy. Common causes:
1. **Service not running**: Check `docker-compose ps` and `docker-compose logs <service>`.
2. **Stale health state**: Kong caches health state. Fully recreate the container:
   ```bash
   docker-compose rm -f -s kong && docker-compose up -d kong
   ```
   Wait ~12 seconds for health checks to pass (5s interval × 2 successes required).

### Kong won't start
Check logs with `docker-compose logs kong`. Common issues:
- Invalid `kong.yml` syntax
- Custom plugin not installed (stick with bundled plugins)
