package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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

// getUserHandler handles the GET /user endpoint.
func getUserHandler(w http.ResponseWriter, r *http.Request, userManager UserManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	users, err := userManager.AllUsers(r.Context())
	if err != nil {
		logger.Error("error in GET /user", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	out := make([]userHandle, len(users))
	for i, u := range users {
		suspendedStatus, err := getSuspendedStatusErrCodeToText(u.SuspendedStatus)
		if err != nil {
			logger.Error("error getting suspended status in GET /user", "err", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		out[i] = userHandle{
			ID:              u.IdentScreenName.String(),
			ScreenName:      u.DisplayScreenName.String(),
			IsICQ:           u.IsICQ,
			SuspendedStatus: suspendedStatus,
			IsBot:           u.IsBot,
		}
	}

	if err := json.NewEncoder(w).Encode(out); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// getUserBuddyIconHandler handles the GET /user/{screenname}/icon endpoint.
func getUserBuddyIconHandler(w http.ResponseWriter, r *http.Request, u UserManager, f FeedBagRetriever, b BARTAssetManager, logger *slog.Logger) {
	screenName := state.NewIdentScreenName(r.PathValue("screenname"))
	user, err := u.User(r.Context(), screenName)
	if err != nil {
		logger.Error("error retrieving user", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	} else if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	iconRef, err := f.BuddyIconMetadata(r.Context(), screenName)
	if err != nil {
		logger.Error("error retrieving buddy icon ref", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	} else if iconRef == nil || iconRef.HasClearIconHash() {
		http.Error(w, "icon not found", http.StatusNotFound)
		return
	}

	icon, err := b.BARTItem(r.Context(), iconRef.Hash)
	if err != nil {
		logger.Error("error retrieving buddy icon bart item", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", http.DetectContentType(icon))
	w.Write(icon)
}

// getUserAccountHandler handles the GET /user/{screenname}/account endpoint.
func getUserAccountHandler(w http.ResponseWriter, r *http.Request, userManager UserManager, a AccountManager, p ProfileRetriever, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	screenName := r.PathValue("screenname")
	user, err := userManager.User(r.Context(), state.NewIdentScreenName(screenName))
	if err != nil {
		logger.Error("error in GET /user/{screenname}/account", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	} else if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	var emailAddress string
	email, err := a.EmailAddress(r.Context(), user.IdentScreenName)
	if err != nil {
		emailAddress = ""
	} else {
		emailAddress = email.String()
	}

	regStatus, err := a.RegStatus(r.Context(), user.IdentScreenName)
	if err != nil {
		logger.Error("error in GET /user/*/account RegStatus", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	confirmStatus, err := a.ConfirmStatus(r.Context(), user.IdentScreenName)
	if err != nil {
		logger.Error("error in GET /user/*/account ConfirmStatus", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	profile, err := p.Profile(r.Context(), user.IdentScreenName)
	if err != nil {
		logger.Error("error in GET /user/*/account Profile", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	suspendedStatusText, err := getSuspendedStatusErrCodeToText(user.SuspendedStatus)
	if err != nil {
		logger.Error("error in GET /user/{screenname}/account", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}

	out := userAccountHandle{
		ID:              user.IdentScreenName.String(),
		ScreenName:      user.DisplayScreenName.String(),
		EmailAddress:    emailAddress,
		RegStatus:       regStatus,
		Confirmed:       confirmStatus,
		Profile:         profile.ProfileText,
		IsICQ:           user.IsICQ,
		SuspendedStatus: suspendedStatusText,
		IsBot:           user.IsBot,
	}
	if err := json.NewEncoder(w).Encode(out); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// putUserPasswordHandler handles the PUT /user/password endpoint.
func putUserPasswordHandler(w http.ResponseWriter, r *http.Request, userManager UserManager, logger *slog.Logger) {
	input, err := userFromBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sn := state.NewIdentScreenName(input.ScreenName)
	if err := userManager.SetUserPassword(r.Context(), sn, input.Password); err != nil {
		switch {
		case errors.Is(err, state.ErrNoUser):
			http.Error(w, "user does not exist", http.StatusNotFound)
			return
		case errors.Is(err, state.ErrPasswordInvalid):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			logger.Error("error updating user password PUT /user/password", "err", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
	_, _ = fmt.Fprintln(w, "Password successfully reset.")
}

// deleteUserHandler handles the DELETE /user endpoint.
func deleteUserHandler(w http.ResponseWriter, r *http.Request, manager UserManager, logger *slog.Logger) {
	user, err := userFromBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = manager.DeleteUser(r.Context(), state.NewIdentScreenName(user.ScreenName))
	switch {
	case errors.Is(err, state.ErrNoUser):
		http.Error(w, "user does not exist", http.StatusNotFound)
		return
	case err != nil:
		logger.Error("error deleting user DELETE /user", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, _ = fmt.Fprintln(w, "User account successfully deleted.")
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

// getSuspendedStatusTextToErrCode maps the given suspendedStatusText to the appropriate error code, or 0x0 for none.
func getSuspendedStatusTextToErrCode(suspendedStatusText string) (uint16, error) {
	suspendedStatusTextMap := map[string]uint16{
		"":              0x0,
		"deleted":       wire.LoginErrDeletedAccount,
		"expired":       wire.LoginErrExpiredAccount,
		"suspended":     wire.LoginErrSuspendedAccount,
		"suspended_age": wire.LoginErrSuspendedAccountAge,
	}
	if suspendedStatus, ok := suspendedStatusTextMap[suspendedStatusText]; !ok {
		return 0x0, errors.New("unable to map suspendedText to error code")
	} else {
		return suspendedStatus, nil
	}
}
