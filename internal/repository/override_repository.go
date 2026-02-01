package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"service-timetable/internal/domain"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type DailyOverrideRepository interface {
	Upsert(ctx context.Context, override domain.DailyOverride) error
	ListByDate(ctx context.Context, classID uuid.UUID, date time.Time) ([]domain.DailyOverride, error)
}

type DailyOverridePostgresRepository struct {
	execer Execer
}

func NewDailyOverridePostgresRepository(execer Execer) *DailyOverridePostgresRepository {
	return &DailyOverridePostgresRepository{execer: execer}
}

func (r *DailyOverridePostgresRepository) Upsert(ctx context.Context, override domain.DailyOverride) error {
	const query = `
INSERT INTO timetable.daily_overrides (
	id,
	class_id,
	date,
	slot_index,
	course_code,
	start_time,
	end_time,
	venue,
	status,
	created_at,
	updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
ON CONFLICT (class_id, date, slot_index)
DO UPDATE SET
	course_code = EXCLUDED.course_code,
	start_time = EXCLUDED.start_time,
	end_time = EXCLUDED.end_time,
	venue = EXCLUDED.venue,
	status = EXCLUDED.status,
	updated_at = now()
`

	_, err := r.execer.ExecContext(
		ctx,
		query,
		override.ID,
		override.ClassID,
		override.Date,
		override.SlotIndex,
		override.CourseCode,
		override.StartTime,
		override.EndTime,
		override.Venue,
		override.Status,
	)
	return err
}

func (r *DailyOverridePostgresRepository) ListByDate(ctx context.Context, classID uuid.UUID, date time.Time) ([]domain.DailyOverride, error) {
	const query = `
SELECT id, class_id, date, slot_index, course_code, start_time, end_time, venue, status
FROM timetable.daily_overrides
WHERE class_id = $1 AND date = $2
ORDER BY slot_index ASC
`

	rows, err := r.execer.QueryContext(ctx, query, classID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []domain.DailyOverride
	for rows.Next() {
		var override domain.DailyOverride
		var startTime sql.NullTime
		var endTime sql.NullTime
		var courseCode sql.NullString
		var venue sql.NullString
		if err := rows.Scan(
			&override.ID,
			&override.ClassID,
			&override.Date,
			&override.SlotIndex,
			&courseCode,
			&startTime,
			&endTime,
			&venue,
			&override.Status,
		); err != nil {
			return nil, err
		}
		if courseCode.Valid {
			override.CourseCode = courseCode.String
		}
		if venue.Valid {
			override.Venue = venue.String
		}
		if startTime.Valid {
			override.StartTime = &startTime.Time
		}
		if endTime.Valid {
			override.EndTime = &endTime.Time
		}
		overrides = append(overrides, override)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return overrides, nil
}
