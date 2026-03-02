package database

import (
	"log"
	"todo_list/config"
	s "todo_list/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() (*gorm.DB, error) {
	dsn := config.LoadDBConfig()
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Could not connect to database %v", err)
	}
	if err := db.AutoMigrate(&s.Todo{},&s.KafkaEvent{}); err != nil {
		log.Fatalf("Could not migrate %v", err)
	}
	return db, nil

}
