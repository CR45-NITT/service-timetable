ALTER TABLE timetable.daily_overrides
    RENAME COLUMN course_name TO course_code;

ALTER TABLE timetable.daily_overrides
    ADD COLUMN IF NOT EXISTS start_time time NULL,
    ADD COLUMN IF NOT EXISTS end_time time NULL,
    ADD COLUMN IF NOT EXISTS venue text NULL;

ALTER TABLE timetable.daily_overrides
    ALTER COLUMN course_code DROP NOT NULL;
