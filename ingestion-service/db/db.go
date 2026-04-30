package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var ORM *gorm.DB

func GetDB() *gorm.DB {
	return ORM
}

func InitDB(user, password, dbName, host, port string) {
	var err error
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbName)
	ORM, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	db, err := ORM.DB()
	if err != nil {
		log.Fatal(err)
	}
	
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	println("Connected to database")
}
