package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"service-timetable/internal/domain"
)

type DefaultSlotRepository interface {
	ListByWeekday(ctx context.Context, classID uuid.UUID, weekday int) ([]domain.DefaultSlot, error)
	ReplaceByWeekday(ctx context.Context, classID uuid.UUID, weekday int, slots []domain.DefaultSlot) error
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
		var startTimeRaw string
		var endTimeRaw string
		if err := rows.Scan(
			&slot.ClassID,
			&slot.Weekday,
			&slot.CourseCode,
			&startTimeRaw,
			&endTimeRaw,
			&slot.Venue,
		); err != nil {
			return nil, err
		}
		startTime, err := parseSlotClockTime(startTimeRaw)
		if err != nil {
			return nil, err
		}
		endTime, err := parseSlotClockTime(endTimeRaw)
		if err != nil {
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

func parseSlotClockTime(raw string) (time.Time, error) {
	if parsed, err := time.Parse("15:04:05", raw); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse("15:04", raw); err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("invalid time value: %q", raw)
}

func (r *DefaultSlotPostgresRepository) ReplaceByWeekday(ctx context.Context, classID uuid.UUID, weekday int, slots []domain.DefaultSlot) error {
	const deleteQuery = `
DELETE FROM timetable.default_slots
WHERE class_id = $1 AND weekday = $2
`

	if _, err := r.execer.ExecContext(ctx, deleteQuery, classID, weekday); err != nil {
		return err
	}

	if len(slots) == 0 {
		return nil
	}

	const insertQuery = `
INSERT INTO timetable.default_slots (
	class_id,
	weekday,
	course_code,
	start_time,
	end_time,
	venue
) VALUES ($1, $2, $3, $4, $5, $6)
`

	for _, slot := range slots {
		if _, err := r.execer.ExecContext(
			ctx,
			insertQuery,
			classID,
			weekday,
			slot.CourseCode,
			slot.StartTime.Format("15:04:05"),
			slot.EndTime.Format("15:04:05"),
			slot.Venue,
		); err != nil {
			return err
		}
	}

	return nil
}
