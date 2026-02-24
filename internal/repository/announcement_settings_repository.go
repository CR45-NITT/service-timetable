package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"service-timetable/internal/domain"
)

type AnnouncementSettingsRepository interface {
	ListAll(ctx context.Context) ([]domain.AnnouncementSettings, error)
	GetByClassID(ctx context.Context, classID uuid.UUID) (domain.AnnouncementSettings, error)
	MarkAnnounced(ctx context.Context, classID uuid.UUID, date time.Time) (bool, error)
	Upsert(ctx context.Context, settings domain.AnnouncementSettings) error
	SetLastAnnouncedDate(ctx context.Context, classID uuid.UUID, date time.Time) error
}

type AnnouncementSettingsPostgresRepository struct {
	execer Execer
}

func NewAnnouncementSettingsPostgresRepository(execer Execer) *AnnouncementSettingsPostgresRepository {
	return &AnnouncementSettingsPostgresRepository{execer: execer}
}

func (r *AnnouncementSettingsPostgresRepository) ListAll(ctx context.Context) ([]domain.AnnouncementSettings, error) {
	const query = `
SELECT class_id, matrix_room_id, daily_announce_time, daily_template, update_template, last_announced_date
FROM timetable.announcement_settings
ORDER BY class_id
`

	rows, err := r.execer.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []domain.AnnouncementSettings
	for rows.Next() {
		var entry domain.AnnouncementSettings
		var lastDate sql.NullTime
		var dailyAnnounceTimeRaw string
		if err := rows.Scan(
			&entry.ClassID,
			&entry.MatrixRoomID,
			&dailyAnnounceTimeRaw,
			&entry.DailyTemplate,
			&entry.UpdateTemplate,
			&lastDate,
		); err != nil {
			return nil, err
		}
		parsedTime, err := parseClockTime(dailyAnnounceTimeRaw)
		if err != nil {
			return nil, err
		}
		entry.DailyAnnounceTime = parsedTime
		if lastDate.Valid {
			entry.LastAnnouncedDate = &lastDate.Time
		}
		settings = append(settings, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *AnnouncementSettingsPostgresRepository) GetByClassID(ctx context.Context, classID uuid.UUID) (domain.AnnouncementSettings, error) {
	const query = `
SELECT class_id, matrix_room_id, daily_announce_time, daily_template, update_template, last_announced_date
FROM timetable.announcement_settings
WHERE class_id = $1
`

	var entry domain.AnnouncementSettings
	var lastDate sql.NullTime
	var dailyAnnounceTimeRaw string
	if err := r.execer.QueryRowContext(ctx, query, classID).Scan(
		&entry.ClassID,
		&entry.MatrixRoomID,
		&dailyAnnounceTimeRaw,
		&entry.DailyTemplate,
		&entry.UpdateTemplate,
		&lastDate,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.AnnouncementSettings{}, err
		}
		return domain.AnnouncementSettings{}, err
	}
	parsedTime, err := parseClockTime(dailyAnnounceTimeRaw)
	if err != nil {
		return domain.AnnouncementSettings{}, err
	}
	entry.DailyAnnounceTime = parsedTime
	if lastDate.Valid {
		entry.LastAnnouncedDate = &lastDate.Time
	}

	return entry, nil
}

func parseClockTime(raw string) (time.Time, error) {
	value := raw
	if parsed, err := time.Parse("15:04:05", value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse("15:04", value); err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("invalid daily_announce_time value: %q", raw)
}

func (r *AnnouncementSettingsPostgresRepository) MarkAnnounced(ctx context.Context, classID uuid.UUID, date time.Time) (bool, error) {
	const query = `
UPDATE timetable.announcement_settings
SET last_announced_date = $2
WHERE class_id = $1
  AND (last_announced_date IS NULL OR last_announced_date < $2)
`

	result, err := r.execer.ExecContext(ctx, query, classID, date)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *AnnouncementSettingsPostgresRepository) Upsert(ctx context.Context, settings domain.AnnouncementSettings) error {
	const query = `
INSERT INTO timetable.announcement_settings (
	class_id,
	matrix_room_id,
	daily_announce_time,
	daily_template,
	update_template,
	last_announced_date
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (class_id)
DO UPDATE SET
	matrix_room_id = EXCLUDED.matrix_room_id,
	daily_announce_time = EXCLUDED.daily_announce_time,
	daily_template = EXCLUDED.daily_template,
	update_template = EXCLUDED.update_template,
	last_announced_date = COALESCE(EXCLUDED.last_announced_date, timetable.announcement_settings.last_announced_date)
`

	_, err := r.execer.ExecContext(
		ctx,
		query,
		settings.ClassID,
		settings.MatrixRoomID,
		settings.DailyAnnounceTime.Format("15:04:05"),
		settings.DailyTemplate,
		settings.UpdateTemplate,
		settings.LastAnnouncedDate,
	)
	return err
}

func (r *AnnouncementSettingsPostgresRepository) SetLastAnnouncedDate(ctx context.Context, classID uuid.UUID, date time.Time) error {
	const query = `
UPDATE timetable.announcement_settings
SET last_announced_date = $2
WHERE class_id = $1
`

	_, err := r.execer.ExecContext(ctx, query, classID, date)
	return err
}
