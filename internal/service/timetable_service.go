package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"service-timetable/internal/domain"
	"service-timetable/internal/repository"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type IdentityClient interface {
	GetMe(ctx context.Context, userID uuid.UUID) (IdentityUser, error)
}

type IdentityUser struct {
	ID    uuid.UUID
	Roles []IdentityRole
}

type IdentityRole struct {
	Name    string
	ClassID *uuid.UUID
}

type TimetableService struct {
	txManager repository.TxManager
	identity  IdentityClient
	clock     func() time.Time
}

func NewTimetableService(txManager repository.TxManager, identity IdentityClient) *TimetableService {
	return &TimetableService{
		txManager: txManager,
		identity:  identity,
		clock:     time.Now,
	}
}

func (s *TimetableService) UpdateTodayOverride(
	ctx context.Context,
	requesterID uuid.UUID,
	classID uuid.UUID,
	slotIndex int,
	courseCode string,
	startTime *time.Time,
	endTime *time.Time,
	venue string,
	status string,
) error {
	date := truncateToDateLocal(s.clock())
	return s.CreateDailyOverride(
		ctx,
		requesterID,
		classID,
		date,
		slotIndex,
		courseCode,
		startTime,
		endTime,
		venue,
		status,
	)
}

func (s *TimetableService) CreateDailyOverride(
	ctx context.Context,
	requesterID uuid.UUID,
	classID uuid.UUID,
	date time.Time,
	slotIndex int,
	courseCode string,
	startTime *time.Time,
	endTime *time.Time,
	venue string,
	status string,
) error {
	if slotIndex <= 0 || !isValidStatus(status) {
		return ErrInvalidInput
	}
	if status != "cancelled" {
		if courseCode == "" || startTime == nil || endTime == nil || venue == "" {
			return ErrInvalidInput
		}
	}

	user, err := s.identity.GetMe(ctx, requesterID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, ErrUnauthorized) {
			return ErrUnauthorized
		}
		return err
	}

	if !isAuthorized(user, classID) {
		return ErrUnauthorized
	}

	localDate := truncateToDateLocal(date)
	override := domain.DailyOverride{
		ID:         uuid.New(),
		ClassID:    classID,
		Date:       localDate,
		SlotIndex:  slotIndex,
		CourseCode: courseCode,
		StartTime:  startTime,
		EndTime:    endTime,
		Venue:      venue,
		Status:     status,
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context, repos repository.TxRepositories) error {
		if err := repos.Overrides.Upsert(ctx, override); err != nil {
			return err
		}

		settings, err := repos.Settings.GetByClassID(ctx, classID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && shouldEmitLateUpdate(settings, localDate, s.clock()) {
			slot, err := s.resolveSingleSlot(ctx, repos, classID, localDate, slotIndex, override)
			if err != nil {
				return err
			}
			payload := domain.TimetableUpdatedPayload{
				ClassID:        classID.String(),
				Date:           localDate.Format("2006-01-02"),
				UpdateTemplate: settings.UpdateTemplate,
				Slots:          []domain.TimetableSlotPayload{slotToPayload(slot)},
				UpdatedBy:      requesterID.String(),
			}

			event := domain.TimetableEvent{
				EventType: "TimetableUpdated",
				Payload:   payload,
			}

			if err := repos.Outbox.Insert(ctx, event); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *TimetableService) ResolveTimetable(ctx context.Context, classID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	var resolved []domain.Slot
	err := s.txManager.WithTx(ctx, func(ctx context.Context, repos repository.TxRepositories) error {
		slots, err := s.resolveTimetableWithRepos(ctx, repos, classID, date)
		if err != nil {
			return err
		}
		resolved = slots
		return nil
	})
	return resolved, err
}

func (s *TimetableService) EmitDailyAnnouncementIfDue(ctx context.Context, now time.Time) error {
	var settings []domain.AnnouncementSettings
	err := s.txManager.WithTx(ctx, func(ctx context.Context, repos repository.TxRepositories) error {
		var err error
		settings, err = repos.Settings.ListAll(ctx)
		return err
	})
	if err != nil {
		return err
	}

	for _, setting := range settings {
		if !isAnnouncementDue(setting, now) {
			continue
		}

		date := truncateToDateLocal(now)
		err := s.txManager.WithTx(ctx, func(ctx context.Context, repos repository.TxRepositories) error {
			marked, err := repos.Settings.MarkAnnounced(ctx, setting.ClassID, date)
			if err != nil {
				return err
			}
			if !marked {
				return nil
			}

			resolved, err := s.resolveTimetableWithRepos(ctx, repos, setting.ClassID, date)
			if err != nil {
				return err
			}

			payload := domain.DailyTimetableAnnouncedPayload{
				ClassID:      setting.ClassID.String(),
				Date:         date.Format("2006-01-02"),
				MatrixRoomID: setting.MatrixRoomID,
				Template:     setting.DailyTemplate,
				Slots:        slotsToPayloads(resolved),
			}

			event := domain.TimetableEvent{
				EventType: "DailyTimetableAnnounced",
				Payload:   payload,
			}

			return repos.Outbox.Insert(ctx, event)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TimetableService) resolveTimetableWithRepos(
	ctx context.Context,
	repos repository.TxRepositories,
	classID uuid.UUID,
	date time.Time,
) ([]domain.Slot, error) {
	weekday := weekdayNumber(date)
	defaults, err := repos.DefaultSlots.ListByWeekday(ctx, classID, weekday)
	if err != nil {
		return nil, err
	}

	overrides, err := repos.Overrides.ListByDate(ctx, classID, truncateToDateLocal(date))
	if err != nil {
		return nil, err
	}

	baseSlots := make([]domain.Slot, 0, len(defaults))
	for idx, def := range defaults {
		slot := domain.Slot{
			SlotIndex:  idx + 1,
			CourseCode: def.CourseCode,
			StartTime:  def.StartTime,
			EndTime:    def.EndTime,
			Venue:      def.Venue,
			Status:     "scheduled",
		}
		baseSlots = append(baseSlots, slot)
	}

	byIndex := make(map[int]domain.Slot, len(baseSlots))
	for _, slot := range baseSlots {
		byIndex[slot.SlotIndex] = slot
	}

	for _, override := range overrides {
		resolved := applyOverride(byIndex[override.SlotIndex], override)
		byIndex[override.SlotIndex] = resolved
	}

	maxIndex := len(baseSlots)
	for _, override := range overrides {
		if override.SlotIndex > maxIndex {
			maxIndex = override.SlotIndex
		}
	}

	resolved := make([]domain.Slot, 0, maxIndex)
	for i := 1; i <= maxIndex; i++ {
		slot, ok := byIndex[i]
		if !ok {
			continue
		}
		resolved = append(resolved, slot)
	}

	return resolved, nil
}

func (s *TimetableService) resolveSingleSlot(
	ctx context.Context,
	repos repository.TxRepositories,
	classID uuid.UUID,
	date time.Time,
	slotIndex int,
	override domain.DailyOverride,
) (domain.Slot, error) {
	weekday := weekdayNumber(date)
	defaults, err := repos.DefaultSlots.ListByWeekday(ctx, classID, weekday)
	if err != nil {
		return domain.Slot{}, err
	}

	var base domain.Slot
	if slotIndex > 0 && slotIndex <= len(defaults) {
		def := defaults[slotIndex-1]
		base = domain.Slot{
			SlotIndex:  slotIndex,
			CourseCode: def.CourseCode,
			StartTime:  def.StartTime,
			EndTime:    def.EndTime,
			Venue:      def.Venue,
			Status:     "scheduled",
		}
	}

	resolved := applyOverride(base, override)
	if resolved.SlotIndex == 0 {
		resolved.SlotIndex = slotIndex
	}

	return resolved, nil
}

func isAuthorized(user IdentityUser, classID uuid.UUID) bool {
	for _, role := range user.Roles {
		switch role.Name {
		case "faculty":
			return true
		case "cr":
			if role.ClassID != nil && *role.ClassID == classID {
				return true
			}
		}
	}
	return false
}

func isValidStatus(status string) bool {
	switch status {
	case "scheduled", "cancelled", "replaced":
		return true
	default:
		return false
	}
}

func truncateToDateLocal(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}

func weekdayNumber(t time.Time) int {
	weekday := t.In(time.Local).Weekday()
	if weekday == time.Sunday {
		return 7
	}
	return int(weekday)
}

func applyOverride(base domain.Slot, override domain.DailyOverride) domain.Slot {
	resolved := base
	resolved.SlotIndex = override.SlotIndex
	resolved.Status = override.Status
	if override.Status == "cancelled" {
		if resolved.CourseCode == "" {
			resolved.CourseCode = override.CourseCode
		}
		if resolved.Venue == "" {
			resolved.Venue = override.Venue
		}
		if override.StartTime != nil {
			resolved.StartTime = *override.StartTime
		}
		if override.EndTime != nil {
			resolved.EndTime = *override.EndTime
		}
		return resolved
	}

	resolved.CourseCode = override.CourseCode
	if override.StartTime != nil {
		resolved.StartTime = *override.StartTime
	}
	if override.EndTime != nil {
		resolved.EndTime = *override.EndTime
	}
	if override.Venue != "" {
		resolved.Venue = override.Venue
	}
	return resolved
}

func slotToPayload(slot domain.Slot) domain.TimetableSlotPayload {
	return domain.TimetableSlotPayload{
		SlotIndex:  slot.SlotIndex,
		CourseCode: slot.CourseCode,
		StartTime:  formatTime(slot.StartTime),
		EndTime:    formatTime(slot.EndTime),
		Venue:      slot.Venue,
		Status:     slot.Status,
	}
}

func slotsToPayloads(slots []domain.Slot) []domain.TimetableSlotPayload {
	result := make([]domain.TimetableSlotPayload, 0, len(slots))
	for _, slot := range slots {
		result = append(result, slotToPayload(slot))
	}
	return result
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04")
}

func isAnnouncementDue(setting domain.AnnouncementSettings, now time.Time) bool {
	localNow := now.In(time.Local)
	announceAt := time.Date(
		localNow.Year(),
		localNow.Month(),
		localNow.Day(),
		setting.DailyAnnounceTime.Hour(),
		setting.DailyAnnounceTime.Minute(),
		setting.DailyAnnounceTime.Second(),
		0,
		localNow.Location(),
	)

	if localNow.Before(announceAt) {
		return false
	}
	if setting.LastAnnouncedDate == nil {
		return true
	}
	return truncateToDateLocal(*setting.LastAnnouncedDate).Before(truncateToDateLocal(localNow))
}

func shouldEmitLateUpdate(setting domain.AnnouncementSettings, date time.Time, now time.Time) bool {
	if setting.LastAnnouncedDate == nil {
		return false
	}
	if truncateToDateLocal(*setting.LastAnnouncedDate) != truncateToDateLocal(date) {
		return false
	}
	localNow := now.In(time.Local)
	announceAt := time.Date(
		localNow.Year(),
		localNow.Month(),
		localNow.Day(),
		setting.DailyAnnounceTime.Hour(),
		setting.DailyAnnounceTime.Minute(),
		setting.DailyAnnounceTime.Second(),
		0,
		localNow.Location(),
	)
	return localNow.After(announceAt)
}
