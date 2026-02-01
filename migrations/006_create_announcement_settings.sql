CREATE TABLE IF NOT EXISTS timetable.announcement_settings (
    class_id uuid PRIMARY KEY,
    matrix_room_id text NOT NULL,
    daily_announce_time time NOT NULL,
    daily_template text NOT NULL,
    update_template text NOT NULL,
    last_announced_date date NULL
);
