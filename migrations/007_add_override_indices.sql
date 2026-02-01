CREATE INDEX IF NOT EXISTS daily_overrides_class_date_slot_idx
    ON timetable.daily_overrides (class_id, date, slot_index);
