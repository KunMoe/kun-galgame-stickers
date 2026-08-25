package database

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"time"

	"kun-galgame-sticker-api/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SanitizeDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	q := u.Query()
	q.Del("schema")
	if q.Get("sslmode") == "" {
		q.Set("sslmode", "disable")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func NewPostgres(cfg config.DatabaseConfig, mode string) *gorm.DB {
	logLevel := logger.Warn
	if mode == "dev" {
		logLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  mode == "dev",
		},
	)

	db, err := gorm.Open(postgres.Open(SanitizeDSN(cfg.URL)), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(fmt.Sprintf("connect postgres: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("sql db: %v", err))
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	slog.Info("database connected")
	return db
}
