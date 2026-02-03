package http

import (
	"log/slog"
	"net/http"
)

type Server struct {
	server http.Server
	logger *slog.Logger
}
