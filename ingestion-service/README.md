# Ingestion Service

Simple Go service for ingestion endpoints.

## Prerequisites

- Go 1.22+ (or any recent stable Go version)

## Setup

From this folder:

```bash
cd ingestion-service
go mod tidy
```

Copy and rename the env template:

```bash
cp .env.example .env
```

## Run

```bash
go run .
```

## Build

```bash
go build ./...
```

## Quick Check

In another terminal:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```
