# Pred

This is the Predictive Maintenance Project of group H for the SE module (CS3023) for CSE batch 23.

## Features

It's a multi-tenant system that allows users to manage their equipment and receive notifications when maintenance is required.

## Project setup

This repository contains all the code to all the services.

### Prerequisites

| Tool           | Version | Install                                                                                                                                                                |
| -------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| git            | >=2.0   | [git-scm.com/downloads](https://git-scm.com/downloads)                                                                                                                 |
| golang         | >=1.23  | [go.dev/doc/install](https://go.dev/doc/install)                                                                                                                       |
| node           | >=20    | [nodejs.org/en/download](https://nodejs.org/en/download)                                                                                                               |
| Docker         | >=20.10 | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)                                                                                                      |
| Docker Compose | >=2.0   | [docs.docker.com/compose/install](https://docs.docker.com/compose/install/). Or Docker Desktop. Or if you are on macOS, you can use [OrbStack](https://orbstack.dev/). |

### Installation

Clone the repository:

```sh
git clone https://github.com/PredictiveOps/Pred.git
cd Pred
```

Start the shared infrastructure (Postgres on host port `5433`, Kafka on `9092`). The Postgres container creates the databases listed in `POSTGRES_MULTIPLE_DATABASES` on first boot:

```sh
docker compose up -d
```

Set up the notifications service:

```sh
cd notifications-service
cp .env.example .env       # edit if defaults don't match
go mod download
go run .
```

Set up the web frontend:

```sh
cd web-frontend
npm install
npm run dev
```

### Running tests

The notifications service ships its own test infrastructure (separate Postgres on host port `5434`) so it doesn't conflict with the dev compose:

```sh
cd notifications-service
make test         # brings up the test Postgres and runs `go test ./...`
make test-down    # tear down when finished
```

## Services

All services must have a `.env.example` file with the required and optional environment variables with their default values.

All services must have a `README.md` file with the following sections:

- What it is supposed to do
- How to run it

Service READMEs must not document database internals (tables, columns, indexes). That level of detail belongs in the code (models, migrations).
