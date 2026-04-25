package main

import (
	"fmt"
	"ingestion-service/config"
	"ingestion-service/db"
	"ingestion-service/handlers"
	"ingestion-service/router"
	"ingestion-service/services"
	"log"
)

func main() {
	config.LoadConfig()

	db.InitDB(config.DBUser, config.DBPassword, config.DBName, config.DBHost, config.DBPort)
	if db.ORM == nil {
		log.Fatal("Database connection not established")
	}

	mqttClient := services.CreateMQTTClient(
		config.MQTTBroker,
		config.MQTTClientID,
		config.MQTTUsername,
		config.MQTTPassword,
	)

	if err := services.ConnectMQTTClient(mqttClient); err != nil {
		log.Fatalf("MQTT connection failed: %v", err)
	}

	if err := services.SubscribeMQTTTopic(mqttClient, config.MQTTTopic, handlers.HandleMQTTMessage); err != nil {
		log.Fatalf("MQTT subscribe failed: %v", err)
	}
	defer services.DisconnectMQTTClient(mqttClient)


	r := router.NewRouter()
	log.Fatal(r.Run(fmt.Sprintf(":%s", config.Port)))
}
