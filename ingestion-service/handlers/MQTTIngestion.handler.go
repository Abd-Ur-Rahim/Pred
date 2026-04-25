package handlers

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func HandleMQTTMessage(_ mqtt.Client, msg mqtt.Message) {
	log.Printf("mqtt message received: topic=%s payload=%s", msg.Topic(), string(msg.Payload()))
}