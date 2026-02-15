// internal/platform/database/postgres.go
package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"payflow/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitializeDatabase handles the complete database setup process
func InitializeDatabase(dsn string) (*gorm.DB, error) {
	// Always try direct connection first (works for Docker and standard local dev)
	if dsn != "" {
		db, err := NewPostgresDB(dsn)
		if err == nil {
			return db, nil
		}
		// Only attempt postgres superuser setup for localhost (local dev with fresh postgres)
		if !strings.Contains(dsn, "@db:") && !strings.Contains(dsn, "host=db") {
			log.Printf("Direct connection failed: %v", err)
		} else {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	// Local dev: try postgres superuser to create user/database if needed
	postgresDSN := "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"

	db, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Printf("Could not connect as postgres user, trying direct connection: %v", err)
		return NewPostgresDB(dsn)
	} else {
		// We're connected as postgres, let's create the user and database
		log.Println("Connected as postgres user, setting up database...")

		// Create user if it doesn't exist
		err = db.Exec(`DO $$ 
			BEGIN 
				IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'payflow_user') THEN
					CREATE USER payflow_user WITH PASSWORD 'payflow_secret';
				END IF;
			END
			$$;`).Error
		if err != nil {
			log.Printf("Warning: Could not create user: %v", err)
		}

		// Check if database exists
		var exists bool
		err = db.Raw(`SELECT EXISTS(SELECT FROM pg_database WHERE datname = 'payflow_db')`).Scan(&exists).Error
		if err != nil {
			log.Printf("Warning: Could not check if database exists: %v", err)
		}

		// Create database if it doesn't exist
		if !exists {
			err = db.Exec(`CREATE DATABASE payflow_db OWNER payflow_user`).Error
			if err != nil {
				log.Printf("Warning: Could not create database: %v", err)
			} else {
				log.Println("Database payflow_db created successfully")
			}
		} else {
			log.Println("Database payflow_db already exists")
		}

		// Grant privileges to the user
		err = db.Exec(`GRANT ALL PRIVILEGES ON DATABASE payflow_db TO payflow_user`).Error
		if err != nil {
			log.Printf("Warning: Could not grant privileges: %v", err)
		}

		// Close the postgres connection
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	// Now connect to our actual database
	db, err = NewPostgresDB(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to payflow_db: %w", err)
	}

	return db, nil
}

// InitializeDatabaseWithAutoMigration handles database setup with optional auto-migration.
// Use this for local development. In production, use InitializeDatabase + traditional migrations.
func InitializeDatabaseWithAutoMigration(dsn string) (*gorm.DB, error) {
	db, err := InitializeDatabase(dsn)
	if err != nil {
		return nil, err
	}

	// Run automigration (only for local development)
	if err := AutoMigrateAll(db); err != nil {
		return nil, fmt.Errorf("failed to run automigration: %w", err)
	}

	return db, nil
}

// NewPostgresDB creates and returns a new GORM DB instance.
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	// GORM's logger configuration
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level (use Warn or Info for more verbosity)
			IgnoreRecordNotFoundError: true,          // Don't log ErrRecordNotFound
			Colorful:                  true,          // Disable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// AutoMigrateAll runs automigration for all domain models
func AutoMigrateAll(db *gorm.DB) error {
	log.Println("Starting automigration...")

	// Create custom types first (ignore 'already exists' errors)
	enumStatements := []string{
		`CREATE TYPE user_role AS ENUM ('admin', 'operator', 'approver')`,
		`CREATE TYPE payroll_status AS ENUM ('draft', 'pending_approval', 'approved', 'processing', 'completed', 'rejected', 'failed')`,
		`CREATE TYPE payroll_entry_detail_type AS ENUM ('earning', 'deduction', 'bonus')`,
	}
	for _, stmt := range enumStatements {
		err := db.Exec(stmt).Error
		if err != nil && !isTypeAlreadyExistsError(err) {
			return err
		}
	}

	// Auto-migrate all domain models
	models := []interface{}{
		&domain.Business{},
		&domain.User{},
		&domain.Cadre{},
		&domain.EarningComponent{},
		&domain.DeductionRule{},
		&domain.Employee{},
		&domain.PayrollRun{},
		&domain.PayrollRunEntry{},
		&domain.PayrollRunEntryDetail{},
		&domain.Transfer{}, // New provider-agnostic transfer model
	}

	for _, model := range models {
		log.Printf("Migrating model: %T", model)
		if err := db.AutoMigrate(model); err != nil {
			log.Printf("Error migrating %T: %v", model, err)
			return err
		}
	}

	log.Println("Automigration completed successfully")
	return nil
}

// isTypeAlreadyExistsError checks if the error is a 'type already exists' error
func isTypeAlreadyExistsError(err error) bool {
	return err != nil && ( // Postgres error code 42710: duplicate_object
	strings.Contains(err.Error(), "already exists") ||
		strings.Contains(err.Error(), "duplicate_object"))
}
