package domain

import (
	"time"

	"github.com/google/uuid"
)

type DefaultSlot struct {
	ClassID    uuid.UUID
	Weekday    int
	CourseCode string
	StartTime  time.Time
	EndTime    time.Time
	Venue      string
}
