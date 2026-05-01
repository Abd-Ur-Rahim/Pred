# Mosquitto Configuration Guide

The MQTT broker is provisioned automatically by the `mosquitto` service in
`docker-compose.yml`. Its entrypoint (`mosquitto/entrypoint.sh`) creates the
password database from environment variables, then starts Mosquitto with
`mosquitto/mosquitto.conf`.

## What gets configured

- **Broker listener:** `0.0.0.0:1883`
- **Anonymous access:** disabled
- **Password database:** generated at `/mosquitto/data/passwords` inside the
  `mosquitto_data` Docker volume
- **ACL file:** `mosquitto/acl`
- **Device user:** `pred-device` / `dev-device-password`
- **Ingestion user:** `pred-ingestion` / `dev-ingestion-password`

## Access rules

| User | Access |
|------|--------|
| `pred-device` | Publish to `devices/+/data` |
| `pred-ingestion` | Subscribe to `devices/+/data` |

## Running it

```sh
docker compose up -d mosquitto
docker compose logs -f mosquitto
```

The broker is available at:

```text
tcp://localhost:1883
```

## Overriding the dev credentials

Set these values in the environment used by Docker Compose, for example a
top-level `.env` file:

```env
MQTT_DEVICE_USERNAME=pred-device
MQTT_DEVICE_PASSWORD=<device-password>
MQTT_INGESTION_USERNAME=pred-ingestion
MQTT_INGESTION_PASSWORD=<ingestion-password>
```

Then recreate the broker:

```sh
docker compose up -d --force-recreate mosquitto
```

Update `ingestion-service/.env` with the ingestion credential:

```env
MQTT_USERNAME=pred-ingestion
MQTT_PASSWORD=<ingestion-password>
```

## Testing credentials

Subscribe as the ingestion service:

```sh
docker compose exec mosquitto mosquitto_sub \
  -h localhost -p 1883 \
  -u pred-ingestion -P dev-ingestion-password \
  -t 'devices/+/data'
```

Publish as a device:

```sh
docker compose exec mosquitto mosquitto_pub \
  -h localhost -p 1883 \
  -u pred-device -P dev-device-password \
  -t 'devices/device-001/data' \
  -m '{"temperature":72.4}'
```

## Troubleshooting

**Connection refused:** check that `docker compose ps mosquitto` shows the
container as healthy and that port `1883` is not already in use.

**Authentication failed:** recreate the broker after changing credential
environment variables. The password database is regenerated on container start.

**Ingestion service cannot connect:** use `tcp://localhost:1883` when running
the ingestion service locally with `go run .`. Use `tcp://mosquitto:1883` only
when the ingestion service runs inside Docker Compose.
