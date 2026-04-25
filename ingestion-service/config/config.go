package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var Port string

func LoadConfig() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	Port = os.Getenv("PORT")

	fmt.Printf("Configuration loaded: PORT=%s\n", Port)
}