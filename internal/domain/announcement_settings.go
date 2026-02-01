package domain

import (
	"time"

	"github.com/google/uuid"
)

type AnnouncementSettings struct {
	ClassID           uuid.UUID
	MatrixRoomID      string
	DailyAnnounceTime time.Time
	DailyTemplate     string
	UpdateTemplate    string
	LastAnnouncedDate *time.Time
}
