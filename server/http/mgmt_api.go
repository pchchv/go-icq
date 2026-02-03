package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
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

// postUserHandler handles the POST /user endpoint.
func postUserHandler(w http.ResponseWriter, r *http.Request, userManager UserManager, newUUID func() uuid.UUID, logger *slog.Logger) {
	input, err := userFromBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sn := state.DisplayScreenName(input.ScreenName)
	if sn.IsUIN() {
		if err := sn.ValidateUIN(); err != nil {
			http.Error(w, fmt.Sprintf("invalid uin: %s", err), http.StatusBadRequest)
			return
		}
	} else if err := sn.ValidateAIMHandle(); err != nil {
		http.Error(w, fmt.Sprintf("invalid screen name: %s", err), http.StatusBadRequest)
		return
	}

	user := state.User{
		AuthKey:           newUUID().String(),
		DisplayScreenName: sn,
		IdentScreenName:   sn.IdentScreenName(),
		IsICQ:             sn.IsUIN(),
	}
	if err := user.HashPassword(input.Password); err != nil {
		http.Error(w, fmt.Sprintf("invalid password: %s", err), http.StatusBadRequest)
		return
	}

	err = userManager.InsertUser(r.Context(), user)
	switch {
	case errors.Is(err, state.ErrDupUser):
		http.Error(w, "user already exists", http.StatusConflict)
		return
	case err != nil:
		logger.Error("error inserting user POST /user", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintln(w, "User account created successfully.")
}

func userFromBody(r *http.Request) (u userWithPassword, err error) {
	u = userWithPassword{}
	if err = json.NewDecoder(r.Body).Decode(&u); err != nil {
		return userWithPassword{}, errors.New("malformed input")
	}
	return
}

// getSuspendedStatusErrCodeToText maps the given suspendedStatus to the appropriate text, or "" for none.
func getSuspendedStatusErrCodeToText(suspendedStatus uint16) (string, error) {
	suspendedStatusTextMap := map[uint16]string{
		0x0:                              "",
		wire.LoginErrDeletedAccount:      "deleted",
		wire.LoginErrExpiredAccount:      "expired",
		wire.LoginErrSuspendedAccount:    "suspended",
		wire.LoginErrSuspendedAccountAge: "suspended_age",
	}
	if st, ok := suspendedStatusTextMap[suspendedStatus]; !ok {
		return "", errors.New("unable to map error code to suspendedText")
	} else {
		return st, nil
	}
}
