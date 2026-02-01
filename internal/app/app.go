package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	transport "service-timetable/internal/http"
	"service-timetable/internal/http/handlers"
	"service-timetable/internal/repository"
	"service-timetable/internal/service"
)

type App struct {
	handler          http.Handler
	timetableService *service.TimetableService
}

func New(db *sql.DB, identityBaseURL string) *App {
	txManager := repository.NewPostgresTxManager(db)
	identityClient := service.NewIdentityHTTPClient(identityBaseURL, service.DefaultIdentityHTTPClient())
	timetableService := service.NewTimetableService(txManager, identityClient)

	adminHandler := handlers.NewAdminHandler(timetableService)
	router := transport.NewRouter(adminHandler)

	return &App{handler: router.Handler(), timetableService: timetableService}
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) EmitDailyAnnouncementIfDue(ctx context.Context, now time.Time) error {
	return a.timetableService.EmitDailyAnnouncementIfDue(ctx, now)
}
