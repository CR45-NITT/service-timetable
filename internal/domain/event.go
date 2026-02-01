package domain

type TimetableEvent struct {
	EventType string
	Payload   any
}

type TimetableSlotPayload struct {
	SlotIndex  int
	CourseCode string
	StartTime  string
	EndTime    string
	Venue      string
	Status     string
}

type DailyTimetableAnnouncedPayload struct {
	ClassID      string
	Date         string
	MatrixRoomID string
	Template     string
	Slots        []TimetableSlotPayload
}

type TimetableUpdatedPayload struct {
	ClassID        string
	Date           string
	UpdateTemplate string
	Slots          []TimetableSlotPayload
	UpdatedBy      string
}
