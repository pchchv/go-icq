package http

import (
	"context"
	"encoding/json"
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

func userFromBody(r *http.Request) (u userWithPassword, err error) {
	u = userWithPassword{}
	if err = json.NewDecoder(r.Body).Decode(&u); err != nil {
		return userWithPassword{}, errors.New("malformed input")
	}
	return
}
