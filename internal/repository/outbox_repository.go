package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"service-timetable/internal/domain"
)

type OutboxRepository interface {
	Insert(ctx context.Context, event domain.TimetableEvent) error
}

type OutboxPostgresRepository struct {
	execer Execer
}

func NewOutboxPostgresRepository(execer Execer) *OutboxPostgresRepository {
	return &OutboxPostgresRepository{execer: execer}
}

func (r *OutboxPostgresRepository) Insert(ctx context.Context, event domain.TimetableEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	const query = `
INSERT INTO timetable.outbox_events (
	id,
	event_type,
	payload,
	created_at,
	published
) VALUES ($1, $2, $3, now(), false)
`

	_, err = r.execer.ExecContext(ctx, query, uuid.New(), event.EventType, payload)
	return err
}
