package handlers

import "log/slog"

const AMF3 AMFVersion = 3

// AMFVersion represents the AMF encoding version.
type AMFVersion int

// AMFEncoder handles AMF encoding operations for WebAPI responses.
type AMFEncoder struct {
	logger *slog.Logger
}

// NewAMFEncoder creates a new AMF encoder instance.
func NewAMFEncoder(logger *slog.Logger) *AMFEncoder {
	return &AMFEncoder{logger: logger}
}
