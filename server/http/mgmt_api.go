package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type Server struct {
	server http.Server
	logger *slog.Logger
}

func (s *Server) ListenAndServe() error {
	s.logger.Info("starting server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("unable to start management API server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	defer s.logger.Info("shutdown complete")
	return s.server.Shutdown(ctx)
}
