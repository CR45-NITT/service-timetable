package repository

import (
	"context"
	"database/sql"
)

type TxRepositories struct {
	Overrides    DailyOverrideRepository
	Outbox       OutboxRepository
	DefaultSlots DefaultSlotRepository
	Settings     AnnouncementSettingsRepository
}

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, repos TxRepositories) error) error
}

type PostgresTxManager struct {
	db *sql.DB
}

func NewPostgresTxManager(db *sql.DB) *PostgresTxManager {
	return &PostgresTxManager{db: db}
}

func (m *PostgresTxManager) WithTx(ctx context.Context, fn func(ctx context.Context, repos TxRepositories) error) error {
	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	repos := TxRepositories{
		Overrides:    NewDailyOverridePostgresRepository(tx),
		Outbox:       NewOutboxPostgresRepository(tx),
		DefaultSlots: NewDefaultSlotPostgresRepository(tx),
		Settings:     NewAnnouncementSettingsPostgresRepository(tx),
	}

	if err := fn(ctx, repos); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}

	return tx.Commit()
}
