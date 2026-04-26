# Pred

This is the Predictive Maintenance Project of group H for the SE module (CS3023) for CSE batch 23.

## Project setup

This repository contains all the code to all the services.

## Services

All services must have a `.env.example` file with the required and optional environment variables with their default values.

### notifications-service

A Go service that consumes messages from a Kafka topic and prints them to stdout.

Run it with:

```sh
cd notifications-service
go run .
```

