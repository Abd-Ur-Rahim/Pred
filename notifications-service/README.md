# Notifications Service

Consumes alert events from a Kafka topic and delivers notifications to end users — either via email or push notification depending on the alert type. Designed as a multi-tenant service where each tenant's notifications are isolated.

## Responsibilities

- **Consume** — reads alert messages from a configured Kafka topic using a consumer group, enabling horizontal scaling
- **Persist** — stores each notification in the database with tenant and recipient context
- **Deliver** — routes to email (SMTP) or device push notification based on the alert's notification type
- **Track** — records delivery status per notification (pending → delivered / failed) for auditability and retry logic

We may use Resend for email delivery.

## Database

Notifications are persisted in PostgreSQL. Schema is managed by GORM via `AutoMigrate` on startup — idempotent, no separate migration tool needed. Models live in the `db` package.

## Multi-tenancy

Every notification is scoped to a tenant. Tenant context is carried in the Kafka message and used to resolve recipient preferences, delivery credentials, and data isolation in the database.

## Configuration

| Variable         | Default                                     | Description                                                                 |
| ---------------- | ------------------------------------------- | --------------------------------------------------------------------------- |
| `KAFKA_BROKERS`  | `localhost:9092`                            | Comma-separated list of Kafka bootstrap brokers                             |
| `KAFKA_TOPIC`    | `notifications`                             | Topic the consumer subscribes to                                            |
| `KAFKA_GROUP_ID` | `notifications-service`                     | Consumer group ID — instances sharing this ID split partitions between them |
| `DATABASE_URL`   | `postgres://localhost:5432/notifications`   | PostgreSQL connection string                                                |

## Running

```sh
cp .env.example .env
# fill in values
go run .
```
