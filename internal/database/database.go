package database

import (
	"fmt"
	"log"
	"posku/internal/config"
	"posku/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.DBHost, config.DBUser, config.DBPass, config.DBName, config.DBPort)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")
}

func Migrate() {
	// Create enum types for PostgreSQL
	DB.Exec(`DO $$ BEGIN CREATE TYPE payment_method AS ENUM ('cash','transfer','wallet'); EXCEPTION WHEN duplicate_object THEN null; END $$;`)
	DB.Exec(`DO $$ BEGIN CREATE TYPE transfer_status AS ENUM ('pending','completed','cancelled'); EXCEPTION WHEN duplicate_object THEN null; END $$;`)

	err := DB.AutoMigrate(
		&models.Company{},
		&models.Store{},
		&models.Warehouse{},
		&models.Product{},
		&models.Stock{},
		&models.Employee{},
		&models.Customer{},
		&models.SalesTransaction{},
		&models.SalesItem{},
		&models.Payment{},
		&models.StockTransfer{},
		&models.TransferItem{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migrated successfully")
}
