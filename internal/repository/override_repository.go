package repository

import (
	"context"
	"database/sql"
	"fmt"
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
		var startTime sql.NullString
		var endTime sql.NullString
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
			parsed, err := parseOptionalClockTime(startTime)
			if err != nil {
				return nil, err
			}
			override.StartTime = parsed
		}
		if endTime.Valid {
			parsed, err := parseOptionalClockTime(endTime)
			if err != nil {
				return nil, err
			}
			override.EndTime = parsed
		}
		overrides = append(overrides, override)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return overrides, nil
}

func parseOptionalClockTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	raw := value.String
	if parsed, err := time.Parse("15:04:05", raw); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse("15:04", raw); err == nil {
		return &parsed, nil
	}
	return nil, fmt.Errorf("invalid time value: %q", raw)
}
