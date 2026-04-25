package main

import (
	"fmt"
	"ingestion-service/config"
	"ingestion-service/db"
	"ingestion-service/router"
	"log"
)

func main() {
	config.LoadConfig()

	db.InitDB(config.DBUser, config.DBPassword, config.DBName, config.DBHost, config.DBPort)
	if db.ORM == nil {
		log.Fatal("Database connection not established")
	}

	r := router.NewRouter()
	log.Fatal(r.Run(fmt.Sprintf(":%s", config.Port)))
}
