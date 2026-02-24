package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"service-timetable/internal/service"
)

type AdminHandler struct {
	service *service.TimetableService
}

func NewAdminHandler(svc *service.TimetableService) *AdminHandler {
	return &AdminHandler{service: svc}
}

func (h *AdminHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/admin/timetable/today", h.handleUpdateToday)
	mux.HandleFunc("/resolved/today", h.handleGetResolvedToday)
	mux.HandleFunc("/admin/announcements/daily/emit", h.handleAnnounceNow)
	mux.HandleFunc("/admin/announcement-settings", h.handleUpdateAnnouncementSettings)
	mux.HandleFunc("/admin/default-slots", h.handleReplaceDefaultSlots)
}

type updateTodayRequest struct {
	ClassID    string `json:"class_id"`
	SlotIndex  int    `json:"slot_index"`
	CourseCode string `json:"course_code"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Venue      string `json:"venue"`
	Status     string `json:"status"`
}

type announceNowRequest struct {
	ClassID string `json:"class_id"`
}

type updateAnnouncementSettingsRequest struct {
	ClassID           string `json:"class_id"`
	MatrixRoomID      string `json:"matrix_room_id"`
	DailyAnnounceTime string `json:"daily_announce_time"`
	DailyTemplate     string `json:"daily_template"`
	UpdateTemplate    string `json:"update_template"`
}

type replaceDefaultSlotsRequest struct {
	ClassID string                    `json:"class_id"`
	Weekday int                       `json:"weekday"`
	Slots   []replaceDefaultSlotInput `json:"slots"`
}

type replaceDefaultSlotInput struct {
	CourseCode string `json:"course_code"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Venue      string `json:"venue"`
}

type resolvedTodayResponse struct {
	ClassID string             `json:"class_id"`
	Date    string             `json:"date"`
	Slots   []resolvedSlotView `json:"slots"`
}

type resolvedSlotView struct {
	SlotIndex  int    `json:"slot_index"`
	CourseCode string `json:"course_code"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Venue      string `json:"venue"`
	Status     string `json:"status"`
}

func (h *AdminHandler) handleUpdateToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requesterID, ok := parseRequesterID(w, r)
	if !ok {
		return
	}

	var req updateTodayRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	startTime, err := parseTimeOptional(req.StartTime)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}
	endTime, err := parseTimeOptional(req.EndTime)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	err = h.service.UpdateTodayOverride(
		r.Context(),
		requesterID,
		classID,
		req.SlotIndex,
		req.CourseCode,
		startTime,
		endTime,
		req.Venue,
		req.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest)
		case errors.Is(err, service.ErrUnauthorized):
			writeError(w, http.StatusForbidden)
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound)
		case errors.Is(err, service.ErrConflict):
			writeError(w, http.StatusConflict)
		default:
			writeError(w, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) handleGetResolvedToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	classIDText := r.URL.Query().Get("class_id")
	classID, err := uuid.Parse(classIDText)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	resolved, err := h.service.GetResolvedTodayPublic(r.Context(), classID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest)
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound)
		default:
			writeError(w, http.StatusInternalServerError)
		}
		return
	}

	resp := resolvedTodayResponse{
		ClassID: resolved.ClassID.String(),
		Date:    resolved.Date.Format("2006-01-02"),
		Slots:   make([]resolvedSlotView, 0, len(resolved.Slots)),
	}
	for _, slot := range resolved.Slots {
		resp.Slots = append(resp.Slots, resolvedSlotView{
			SlotIndex:  slot.SlotIndex,
			CourseCode: slot.CourseCode,
			StartTime:  formatTime(slot.StartTime),
			EndTime:    formatTime(slot.EndTime),
			Venue:      slot.Venue,
			Status:     slot.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AdminHandler) handleAnnounceNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requesterID, ok := parseRequesterID(w, r)
	if !ok {
		return
	}

	var req announceNowRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	if err := h.service.AnnounceNow(r.Context(), requesterID, classID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest)
		case errors.Is(err, service.ErrUnauthorized):
			writeError(w, http.StatusForbidden)
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound)
		default:
			writeError(w, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) handleUpdateAnnouncementSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requesterID, ok := parseRequesterID(w, r)
	if !ok {
		return
	}

	var req updateAnnouncementSettingsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	announceTime, err := parseRequiredClockTime(req.DailyAnnounceTime)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	err = h.service.UpdateAnnouncementSettings(
		r.Context(),
		requesterID,
		classID,
		req.MatrixRoomID,
		announceTime,
		req.DailyTemplate,
		req.UpdateTemplate,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest)
		case errors.Is(err, service.ErrUnauthorized):
			writeError(w, http.StatusForbidden)
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound)
		default:
			writeError(w, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) handleReplaceDefaultSlots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requesterID, ok := parseRequesterID(w, r)
	if !ok {
		return
	}

	var req replaceDefaultSlotsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return
	}

	slots := make([]service.DefaultSlotInput, 0, len(req.Slots))
	for _, item := range req.Slots {
		startTime, err := parseRequiredClockTime(item.StartTime)
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		endTime, err := parseRequiredClockTime(item.EndTime)
		if err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}

		slots = append(slots, service.DefaultSlotInput{
			CourseCode: item.CourseCode,
			StartTime:  startTime,
			EndTime:    endTime,
			Venue:      item.Venue,
		})
	}

	err = h.service.ReplaceDefaultSlots(r.Context(), requesterID, classID, req.Weekday, slots)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest)
		case errors.Is(err, service.ErrUnauthorized):
			writeError(w, http.StatusForbidden)
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound)
		default:
			writeError(w, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte("{}"))
}

func parseTimeOptional(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("15:04", value, time.Local)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseRequiredClockTime(value string) (time.Time, error) {
	parsed, err := time.ParseInLocation("15:04", value, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func parseRequesterID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userIDHeader := r.Header.Get("X-User-ID")
	if userIDHeader == "" {
		writeError(w, http.StatusBadRequest)
		return uuid.Nil, false
	}
	requesterID, err := uuid.Parse(userIDHeader)
	if err != nil {
		writeError(w, http.StatusBadRequest)
		return uuid.Nil, false
	}
	return requesterID, true
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("15:04")
}
