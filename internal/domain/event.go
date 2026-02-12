package domain

type TimetableEvent struct {
	EventType string `json:"event_type"`
	Payload   any    `json:"payload"`
}

type TimetableSlotPayload struct {
	SlotIndex  int    `json:"slot_index"`
	CourseCode string `json:"course_code"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Venue      string `json:"venue"`
	Status     string `json:"status"`
}

type DailyTimetableAnnouncedPayload struct {
	ClassID      string                 `json:"class_id"`
	Date         string                 `json:"date"`
	MatrixRoomID string                 `json:"matrix_room_id"`
	Template     string                 `json:"template"`
	Slots        []TimetableSlotPayload `json:"slots"`
}

type TimetableUpdatedPayload struct {
	ClassID        string                 `json:"class_id"`
	Date           string                 `json:"date"`
	MatrixRoomID   string                 `json:"matrix_room_id"`
	UpdateTemplate string                 `json:"update_template"`
	Slots          []TimetableSlotPayload `json:"slots"`
	UpdatedBy      string                 `json:"updated_by"`
}
