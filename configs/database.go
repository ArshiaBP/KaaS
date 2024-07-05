package configs

import (
	"KaaS/models"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
)

var (
	DB *gorm.DB
)

func ConnectToDatabase() {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("DATABASE_HOST"),
			os.Getenv("DATABASE_PORT"),
			os.Getenv("DATABASE_USERNAME"),
			os.Getenv("DATABASE_PASSWORD"),
			os.Getenv("DATABASE_DB"),
		),
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	err = db.AutoMigrate(&models.HealthCheck{})
	if err != nil {
		log.Fatalf("Failed to migrate user tabel: %v", err)
	}
	DB = db
}
