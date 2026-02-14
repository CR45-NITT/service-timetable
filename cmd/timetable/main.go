package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"service-timetable/internal/app"
	servicemigrations "service-timetable/migrations"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	config, err := loadConfig()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}

	debugEnabled := strings.EqualFold(strings.TrimSpace(config.LogLevel), "debug")
	debugf := func(format string, args ...any) {
		if debugEnabled {
			logger.Printf("[DEBUG] "+format, args...)
		}
	}

	debugf("config loaded: http_addr=%s identity_base_url=%s db_max_open=%d db_max_idle=%d db_conn_max_lifetime=%s",
		config.HTTPAddr,
		config.IdentityBaseURL,
		config.DBMaxOpenConns,
		config.DBMaxIdleConns,
		config.DBConnMaxLifetime,
	)

	db, err := sql.Open("pgx", config.DatabaseURL)
	if err != nil {
		logger.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(config.DBMaxOpenConns)
	db.SetMaxIdleConns(config.DBMaxIdleConns)
	db.SetConnMaxLifetime(config.DBConnMaxLifetime)

	if err := db.Ping(); err != nil {
		logger.Fatalf("failed to connect to database: %v", err)
	}
	debugf("database connection successful")

	if err := servicemigrations.Up(db); err != nil {
		logger.Fatalf("failed to run migrations: %v", err)
	}
	debugf("migrations completed successfully")

	application := app.New(db, config.IdentityBaseURL)
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startAnnouncementLoop(shutdownCtx, application, logger)

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
			logger.Printf("http shutdown error: %v", err)
		}
	}()

	logger.Printf("service-timetable listening on %s", config.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("http server error: %v", err)
	}
}

func startAnnouncementLoop(ctx context.Context, application *app.App, logger *log.Logger) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		defer ticker.Stop()
		if err := application.EmitDailyAnnouncementIfDue(context.Background(), time.Now()); err != nil {
			logger.Printf("announcement tick error: %v", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := application.EmitDailyAnnouncementIfDue(context.Background(), now); err != nil {
					logger.Printf("announcement tick error: %v", err)
				}
			}
		}
	}()
}

type config struct {
	DatabaseURL       string
	HTTPAddr          string
	LogLevel          string
	IdentityBaseURL   string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

func loadConfig() (config, error) {
	var cfg config

	var err error
	if cfg.DatabaseURL, err = getRequiredEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}
	cfg.HTTPAddr = getEnv("HTTP_ADDR", ":8080")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	if cfg.IdentityBaseURL, err = getRequiredEnv("IDENTITY_BASE_URL"); err != nil {
		return cfg, err
	}
	if cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 10); err != nil {
		return cfg, err
	}
	if cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		return cfg, err
	}
	if cfg.DBConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func getRequiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", &configError{message: "missing required environment variable: " + key}
	}
	return value, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, &configError{message: "invalid int for " + key + ": " + err.Error()}
	}
	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, &configError{message: "invalid duration for " + key + ": " + err.Error()}
	}
	return parsed, nil
}

type configError struct {
	message string
}

func (e *configError) Error() string {
	return e.message
}

var _ error = (*configError)(nil)
