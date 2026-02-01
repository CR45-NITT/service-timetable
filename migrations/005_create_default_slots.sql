CREATE TABLE IF NOT EXISTS timetable.default_slots (
    class_id uuid NOT NULL,
    weekday integer NOT NULL,
    course_code text NOT NULL,
    start_time time NOT NULL,
    end_time time NOT NULL,
    venue text NOT NULL,
    PRIMARY KEY (class_id, weekday, start_time)
);

CREATE INDEX IF NOT EXISTS default_slots_class_weekday_idx
    ON timetable.default_slots (class_id, weekday);
