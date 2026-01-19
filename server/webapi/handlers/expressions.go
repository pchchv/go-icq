package handlers

import "log/slog"

// ExpressionsHandler handles Web AIM API expressions/buddy icon endpoints.
type ExpressionsHandler struct {
	Logger *slog.Logger
}

// NewExpressionsHandler creates a new ExpressionsHandler.
func NewExpressionsHandler(logger *slog.Logger) *ExpressionsHandler {
	return &ExpressionsHandler{
		Logger: logger,
	}
}
