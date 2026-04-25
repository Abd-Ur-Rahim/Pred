package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var Port, DBHost, DBPort, DBUser, DBPassword, DBName string

func LoadConfig() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	Port = os.Getenv("PORT")
	DBHost = os.Getenv("DB_HOST")
	DBPort = os.Getenv("DB_PORT")
	DBUser = os.Getenv("DB_USER")
	DBPassword = os.Getenv("DB_PASSWORD")
	DBName = os.Getenv("DB_NAME")

	fmt.Printf("Configuration loaded: PORT=%s\n", Port)
}