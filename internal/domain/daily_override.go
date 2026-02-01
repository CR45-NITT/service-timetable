package domain

import (
	"time"

	"github.com/google/uuid"
)

type DailyOverride struct {
	ID         uuid.UUID
	ClassID    uuid.UUID
	Date       time.Time
	SlotIndex  int
	CourseCode string
	StartTime  *time.Time
	EndTime    *time.Time
	Venue      string
	Status     string
}
