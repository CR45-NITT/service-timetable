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

func (h *AdminHandler) handleUpdateToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userIDHeader := r.Header.Get("X-User-ID")
	if userIDHeader == "" {
		writeError(w, http.StatusBadRequest)
		return
	}
	requesterID, err := uuid.Parse(userIDHeader)
	if err != nil {
		writeError(w, http.StatusBadRequest)
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
