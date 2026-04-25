package main

import (
	"fmt"
	"ingestion-service/config"
	"ingestion-service/router"
	"log"
)

func main() {
	config.LoadConfig()

	r := router.NewRouter()
	log.Fatal(r.Run(fmt.Sprintf(":%s", config.Port)))
}
