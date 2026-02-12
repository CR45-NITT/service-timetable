# service-timetable

Production-grade timetable service for CR45. Owns baseline timetable data, today-only overrides, and emits timetable change events via the outbox table.

## Responsibilities

- Manage daily overrides for the timetable.
- Emit timetable change events using an outbox table.
- Authorize requests using `service-identity`.

## Non-responsibilities

- User identity, roles, and contact data.
- Course metadata.
- Attendance tracking.
- Notifications (only emits events).

## Requirements

- Go 1.21+
- PostgreSQL

## Configuration

| Environment Variable | Required | Description |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL DSN (pgx driver). |
| `IDENTITY_BASE_URL` | Yes | Base URL for `service-identity`. |
| `HTTP_ADDR` | No | HTTP bind address. Default: `:8080`. |
| `SHUTDOWN_TIMEOUT` | No | Graceful shutdown timeout. Default: `10s`. |
| `HTTP_READ_TIMEOUT` | No | Read timeout. Default: `5s`. |
| `HTTP_WRITE_TIMEOUT` | No | Write timeout. Default: `10s`. |
| `HTTP_IDLE_TIMEOUT` | No | Idle timeout. Default: `60s`. |

## HTTP

### POST /admin/timetable/today

Headers:

- `X-User-ID: <UUID>`

Body:

```
{
	"class_id": "uuid",
	"slot_index": 3,
	"course_code": "EC301",
	"start_time": "09:00",
	"end_time": "09:50",
	"venue": "E-205",
	"status": "cancelled"
}
```

Rules:

- `status` must be one of: `scheduled`, `cancelled`, `replaced`
- if `status != cancelled`, `course_code`, `start_time`, `end_time`, and `venue` are required

Responses:

- `204 No Content`: override accepted
- `400 Bad Request`: invalid header/body/time format/input
- `403 Forbidden`: requester is not authorized for class
- `404 Not Found`: requester or referenced entity not found
- `409 Conflict`: conflicting change
- `405 Method Not Allowed`: wrong HTTP method
- `500 Internal Server Error`: unexpected error

## Route Inventory

This service currently exposes one HTTP route:

- `POST /admin/timetable/today`

## Migrations

SQL migrations live in [migrations](migrations).

## Default timetable and announcements

Default slots and announcement settings are managed directly in Postgres:

- `timetable.default_slots`
- `timetable.announcement_settings`

## Local development

Use docker-compose for PostgreSQL and service wiring:

- [docker-compose.yml](docker-compose.yml)
