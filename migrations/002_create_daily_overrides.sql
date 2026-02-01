CREATE TABLE IF NOT EXISTS timetable.daily_overrides (
    id uuid PRIMARY KEY,
    class_id uuid NOT NULL,
    date date NOT NULL,
    slot_index integer NOT NULL,
    course_name text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (class_id, date, slot_index)
);

CREATE INDEX IF NOT EXISTS daily_overrides_class_date_idx
    ON timetable.daily_overrides (class_id, date);
