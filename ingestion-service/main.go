package main

import (
	"context"
	"fmt"
	"ingestion-service/config"
	"ingestion-service/db"
	"ingestion-service/handlers"
	"ingestion-service/router"
	"ingestion-service/services"
	"log"
	"time"
)

func main() {
	config.LoadConfig()

	if config.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	gdb, err := db.Open(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	if gdb == nil {
		log.Fatal("Database connection not established")
	}

	// Migrate the schema
	if err := gdb.AutoMigrate(&db.Device{}); err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	kafkaProducer := services.NewKafkaProducer(config.KafkaBrokers, config.KafkaTopic)
	handlers.SetKafkaProducer(kafkaProducer)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Printf("Failed to close Kafka producer: %v", err)
		}
	}()

	pubKeyTTL, err := time.ParseDuration(config.RedisPubKeyTTL)
	if err != nil {
		log.Fatalf("Invalid REDIS_PUBKEY_TTL: %v", err)
	}
	nonceTTL, err := time.ParseDuration(config.RedisNonceTTL)
	if err != nil {
		log.Fatalf("Invalid REDIS_NONCE_TTL: %v", err)
	}

	redisCache, err := services.NewRedisCache(
		config.RedisAddr,
		config.RedisPassword,
		config.RedisDB,
		pubKeyTTL,
		nonceTTL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}
	handlers.SetRedisCache(redisCache)
	defer func() {
		if err := redisCache.Close(); err != nil {
			log.Printf("Failed to close Redis cache: %v", err)
		}
	}()

	mqttClient, err := services.CreateMQTTClient(
		config.MQTTBroker,
		config.MQTTClientID,
		config.MQTTUsername,
		config.MQTTPassword,
		config.MQTTCACert,
	)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	if err := services.ConnectMQTTClient(mqttClient); err != nil {
		log.Fatalf("MQTT connection failed: %v", err)
	}

	handlers.SetRegistrationResponseTopicTemplate(config.MQTTDeviceRegistrationResponseTopic)
	if err := services.SubscribeMQTTTopic(mqttClient, "devices/+/+", handlers.HandleMQTTMessage); err != nil {
		log.Fatalf("MQTT subscribe failed: %v", err)
	}
	defer services.DisconnectMQTTClient(mqttClient)

	r := router.NewRouter(gdb)
	log.Fatal(r.Run(fmt.Sprintf(":%s", config.Port)))
}
