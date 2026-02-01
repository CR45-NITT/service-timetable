CREATE TABLE IF NOT EXISTS timetable.outbox_events (
    id uuid PRIMARY KEY,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published boolean NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS outbox_events_unpublished_idx
    ON timetable.outbox_events (published, created_at)
    WHERE published = false;
