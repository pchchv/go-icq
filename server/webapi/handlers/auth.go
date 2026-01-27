package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/pchchv/go-icq/state"
)

// TokenStore manages authentication tokens.
type TokenStore interface {
	// StoreToken saves an authentication token for a user
	StoreToken(ctx context.Context, token string, screenName state.IdentScreenName, expiresAt time.Time) error
	// ValidateToken checks if a token is valid and returns the associated screen name
	ValidateToken(ctx context.Context, token string) (state.IdentScreenName, error)
	// DeleteToken removes a token
	DeleteToken(ctx context.Context, token string) error
}

// UserManager defines methods for user authentication.
type UserManager interface {
	// AuthenticateUser verifies username and password.
	AuthenticateUser(ctx context.Context, username, password string) (*state.User, error)
	// FindUserByScreenName finds a user by their screen name.
	FindUserByScreenName(ctx context.Context, screenName state.IdentScreenName) (*state.User, error)
	// InsertUser creates a new user (for DISABLE_AUTH mode).
	InsertUser(ctx context.Context, u state.User) error
}

// ClientLoginRequest represents the request body for clientLogin.
type ClientLoginRequest struct {
	DevID    string `json:"devId"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthHandler handles Web AIM API authentication endpoints.
type AuthHandler struct {
	DisableAuth bool
	UserManager UserManager
	TokenStore  TokenStore
	Logger      *slog.Logger
}

// ClientLogin handles POST /auth/clientLogin requests.
// This endpoint authenticates users and returns an authentication token.
func (h *AuthHandler) ClientLogin(w http.ResponseWriter, r *http.Request) {
	var username, password, devID string
	// check Content-Type to determine how to parse the request
	if contentType := r.Header.Get("Content-Type"); contentType == "application/json" {
		// parse JSON body
		var req ClientLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.Logger.Error("failed to parse JSON clientLogin request", "error", err)
			SendError(w, http.StatusBadRequest, "invalid JSON format")
			return
		}

		username = req.Username
		password = req.Password
		devID = req.DevID
	} else {
		// parse form-encoded or URL parameters
		if err := r.ParseForm(); err != nil {
			h.Logger.Error("failed to parse form data", "error", err)
			SendError(w, http.StatusBadRequest, "invalid form data")
			return
		}

		// try form values first, then fall back to query parameters
		username = r.FormValue("s")
		if username == "" {
			username = r.FormValue("username")
		}

		password = r.FormValue("pwd")
		if password == "" {
			password = r.FormValue("password")
		}

		devID = r.FormValue("devId")
		h.Logger.Debug("form-encoded login attempt",
			"username", username,
			"has_password", password != "",
			"devId", devID,
			"form", r.Form)
	}

	// validate required fields
	if username == "" || password == "" {
		SendError(w, http.StatusBadRequest, "username and password required")
		return
	}

	// authenticate user
	user, err := h.UserManager.AuthenticateUser(r.Context(), username, password)
	if err != nil {
		// if DISABLE_AUTH is enabled and user doesn't exist, create the user
		if h.DisableAuth && err.Error() == "user not found" {
			h.Logger.Info("DISABLE_AUTH: Creating new user", "username", username)
			// create new user with the provided username
			newUser := state.User{
				IdentScreenName:   state.NewIdentScreenName(username),
				DisplayScreenName: state.DisplayScreenName(username),
			}

			// insert the new user
			if err := h.UserManager.InsertUser(r.Context(), newUser); err != nil {
				h.Logger.Error("failed to create user", "username", username, "error", err)
				SendError(w, http.StatusInternalServerError, "failed to create user")
				return
			}

			// try to authenticate again after creating the user
			user, err = h.UserManager.AuthenticateUser(r.Context(), username, password)
			if err != nil {
				h.Logger.Error("failed to authenticate after creating user", "username", username, "error", err)
				SendError(w, http.StatusInternalServerError, "internal server error")
				return
			}
		} else {
			h.Logger.Warn("authentication failed", "username", username, "error", err)
			SendError(w, http.StatusUnauthorized, "authentication failed")
			return
		}
	}

	// generate authentication token
	token, err := h.generateToken()
	if err != nil {
		h.Logger.Error("failed to generate token", "error", err)
		SendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// store token with 24 hour expiry
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := h.TokenStore.StoreToken(r.Context(), token, user.IdentScreenName, expiresAt); err != nil {
		h.Logger.Error("failed to store token", "error", err)
		SendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// generate session secret (for signing subsequent requests)
	sessionSecret, err := h.generateToken()
	if err != nil {
		h.Logger.Error("failed to generate session secret", "error", err)
		SendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// build response
	resp := BaseResponse{}
	resp.Response.StatusCode = 200
	resp.Response.StatusText = "OK"
	resp.Response.Data = map[string]interface{}{
		"token": map[string]interface{}{
			"a":         token,
			"expiresIn": 86400, // 24 hours in seconds
		},
		"loginId":        string(user.DisplayScreenName),
		"screenName":     string(user.DisplayScreenName),
		"sessionSecret":  sessionSecret,
		"hostTime":       time.Now().Unix(),
		"tokenExpiresIn": 86400, // 24 hours in seconds
	}

	// send response in requested format (JSON, JSONP, XML, or AMF)
	SendResponse(w, r, resp, h.Logger)
	h.Logger.Info("user authenticated successfully", "username", username, "screenName", user.DisplayScreenName)
}

// generateToken generates a secure random token.
func (h *AuthHandler) generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
