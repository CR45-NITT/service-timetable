package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"service-timetable/internal/app"
)

func main() {
	config := loadConfig()

	db, err := sql.Open("pgx", config.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(config.DBMaxOpenConns)
	db.SetMaxIdleConns(config.DBMaxIdleConns)
	db.SetConnMaxLifetime(config.DBConnMaxLifetime)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	application := app.New(db, config.IdentityBaseURL)
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startAnnouncementLoop(shutdownCtx, application)

	server := &http.Server{
		Addr:              config.HTTPAddr,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("http shutdown error: %v", err)
		}
	}()

	log.Printf("service-timetable listening on %s", config.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server error: %v", err)
	}
}

func startAnnouncementLoop(ctx context.Context, application *app.App) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		defer ticker.Stop()
		if err := application.EmitDailyAnnouncementIfDue(context.Background(), time.Now()); err != nil {
			log.Printf("announcement tick error: %v", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := application.EmitDailyAnnouncementIfDue(context.Background(), now); err != nil {
					log.Printf("announcement tick error: %v", err)
				}
			}
		}
	}()
}

type config struct {
	DatabaseURL       string
	HTTPAddr          string
	IdentityBaseURL   string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

func loadConfig() config {
	return config{
		DatabaseURL:       mustEnv("DATABASE_URL"),
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		IdentityBaseURL:   mustEnv("IDENTITY_BASE_URL"),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing required environment variable: %s", key)
	}
	return value
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("invalid int for %s: %v", key, err)
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("invalid duration for %s: %v", key, err)
	}
	return parsed
}
