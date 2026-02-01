package domain

import "time"

type Slot struct {
	SlotIndex  int
	CourseCode string
	StartTime  time.Time
	EndTime    time.Time
	Venue      string
	Status     string
}
