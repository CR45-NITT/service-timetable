package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"service-timetable/internal/domain"
)

type DefaultSlotRepository interface {
	ListByWeekday(ctx context.Context, classID uuid.UUID, weekday int) ([]domain.DefaultSlot, error)
}

type DefaultSlotPostgresRepository struct {
	execer Execer
}

func NewDefaultSlotPostgresRepository(execer Execer) *DefaultSlotPostgresRepository {
	return &DefaultSlotPostgresRepository{execer: execer}
}

func (r *DefaultSlotPostgresRepository) ListByWeekday(ctx context.Context, classID uuid.UUID, weekday int) ([]domain.DefaultSlot, error) {
	const query = `
SELECT class_id, weekday, course_code, start_time, end_time, venue
FROM timetable.default_slots
WHERE class_id = $1 AND weekday = $2
ORDER BY start_time ASC
`

	rows, err := r.execer.QueryContext(ctx, query, classID, weekday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []domain.DefaultSlot
	for rows.Next() {
		var slot domain.DefaultSlot
		var startTime time.Time
		var endTime time.Time
		if err := rows.Scan(
			&slot.ClassID,
			&slot.Weekday,
			&slot.CourseCode,
			&startTime,
			&endTime,
			&slot.Venue,
		); err != nil {
			return nil, err
		}
		slot.StartTime = startTime
		slot.EndTime = endTime
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return slots, nil
}
