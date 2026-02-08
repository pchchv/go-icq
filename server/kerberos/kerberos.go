package kuberos

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/sync/errgroup"
)

// Server hosts an HTTP endpoint capable of
// handling AIM-style Kerberos authentication.
// The messages are structured as SNACs transmitted over HTTP.
type Server struct {
	servers []*http.Server
	logger  *slog.Logger
}

func (s *Server) ListenAndServe() error {
	if len(s.servers) == 0 {
		s.logger.Debug("no kerberos listeners defined")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	for _, server := range s.servers {
		g.Go(func() error {
			s.logger.Info("starting server", "addr", server.Addr)
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				cancel()
				return fmt.Errorf("unable to start kerberos server: %w", err)
			}
			return nil
		})
	}

	return g.Wait()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if len(s.servers) > 0 {
		for _, srv := range s.servers {
			_ = srv.Shutdown(ctx)
		}
		s.logger.Info("shutdown complete")
	}
	return nil
}
