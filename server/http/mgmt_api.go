package http

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// BARTAsset represents a BART asset entry.
type BARTAsset struct {
	Hash string `json:"hash"`
	Type uint16 `json:"type"`
}

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

// postBARTHandler handles the POST /bart endpoint.
func postBARTHandler(w http.ResponseWriter, r *http.Request, bartAssetManager BARTAssetManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	// extract hash from URL path
	hashStr := r.PathValue("hash")
	if hashStr == "" {
		errorMsg(w, "hash path parameter is required", http.StatusBadRequest)
		return
	}

	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		errorMsg(w, "invalid hash format", http.StatusBadRequest)
		return
	}

	typeStr := r.URL.Query().Get("type")
	if typeStr == "" {
		errorMsg(w, "type query parameter is required", http.StatusBadRequest)
		return
	}

	typeVal, err := strconv.ParseUint(typeStr, 10, 16)
	if err != nil {
		errorMsg(w, "invalid type ID", http.StatusBadRequest)
		return
	}

	bartType := uint16(typeVal)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		errorMsg(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if err := bartAssetManager.InsertBARTItem(r.Context(), hashBytes, data, bartType); err != nil {
		if errors.Is(err, state.ErrBARTItemExists) {
			errorMsg(w, "BART asset already exists", http.StatusConflict)
		} else {
			logger.Error("error in POST /bart", "err", err.Error())
			errorMsg(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := BARTAsset{
		Hash: hex.EncodeToString(hashBytes),
		Type: bartType,
	}
	json.NewEncoder(w).Encode(response)
}

// getBARTByTypeHandler handles the GET /bart endpoint.
func getBARTByTypeHandler(w http.ResponseWriter, r *http.Request, bartAssetManager BARTAssetManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	// get type from query parameter (required)
	typeStr := r.URL.Query().Get("type")
	if typeStr == "" {
		errorMsg(w, "type query parameter is required", http.StatusBadRequest)
		return
	}

	typeVal, err := strconv.ParseUint(typeStr, 10, 16)
	if err != nil {
		errorMsg(w, "invalid type ID", http.StatusBadRequest)
		return
	}

	itemType := uint16(typeVal)
	// get BART items, filtered by type
	items, err := bartAssetManager.ListBARTItems(r.Context(), itemType)
	if err != nil {
		logger.Error("error listing BART items", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// convert to BARTAsset format
	assets := make([]BARTAsset, 0, len(items))
	for _, item := range items {
		assets = append(assets, BARTAsset{
			Hash: item.Hash,
			Type: item.Type,
		})
	}

	if err := json.NewEncoder(w).Encode(assets); err != nil {
		logger.Error("error encoding response", "err", err.Error())
	}
}

// getBARTHandler handles the GET /bart/{hash} endpoint.
func getBARTHandler(w http.ResponseWriter, r *http.Request, bartAssetManager BARTAssetManager, logger *slog.Logger) {
	hashStr := r.PathValue("hash")
	if hashStr == "" {
		errorMsg(w, "hash is required", http.StatusBadRequest)
		return
	}

	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		errorMsg(w, "invalid hash format", http.StatusBadRequest)
		return
	}

	body, err := bartAssetManager.BARTItem(r.Context(), hashBytes)
	if err != nil {
		logger.Error("error retrieving BART asset", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		errorMsg(w, "BART asset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
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

// patchUserAccountHandler handles the PATCH /user/{screenname}/account endpoint.
func patchUserAccountHandler(w http.ResponseWriter, r *http.Request, userManager UserManager, a AccountManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	screenName := r.PathValue("screenname")
	user, err := userManager.User(r.Context(), state.NewIdentScreenName(screenName))
	if err != nil {
		logger.Error("error in PATCH /user/{screenname}/account", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	} else if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	input := userAccountPatch{}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&input); err != nil {
		errorMsg(w, err.Error(), http.StatusBadRequest)
		return
	}

	var modifiedUser bool
	if input.SuspendedStatusText != nil {
		switch *input.SuspendedStatusText {
		case "", "deleted", "expired", "suspended", "suspended_age":
			suspendedStatus, err := getSuspendedStatusTextToErrCode(*input.SuspendedStatusText)
			if err != nil {
				logger.Error("error in PATCH /user/{screenname}/account", "err", err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			} else if suspendedStatus != user.SuspendedStatus {
				if err := a.UpdateSuspendedStatus(r.Context(), suspendedStatus, user.IdentScreenName); err != nil {
					logger.Error("error in PATCH /user/{screenname}/account", "err", err.Error())
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				modifiedUser = true
			}
		default:
			errorMsg(w, "suspended_status must be empty str or one of deleted,expired,suspended,suspended_age", http.StatusBadRequest)
			return
		}
	}

	if input.IsBot != nil && user.IsBot != *input.IsBot {
		if err := a.SetBotStatus(r.Context(), *input.IsBot, user.IdentScreenName); err != nil {
			logger.Error("error in PATCH /user/{screenname}/account", "err", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		modifiedUser = true
	}

	if !modifiedUser {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

// getSessionHandler handles GET /session endpoint.
func getSessionHandler(w http.ResponseWriter, r *http.Request, sessionRetriever SessionRetriever, nowFn func() time.Time) {
	var allSessions []*state.Session
	w.Header().Set("Content-Type", "application/json")
	if screenName := r.PathValue("screenname"); screenName != "" {
		session := sessionRetriever.RetrieveSession(state.NewIdentScreenName(screenName))
		if session == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		allSessions = append(allSessions, session)
	} else {
		allSessions = sessionRetriever.AllSessions()
	}

	ou := onlineUsers{
		Count:    len(allSessions),
		Sessions: make([]sessionHandle, len(allSessions)),
	}
	for i, s := range allSessions {
		instances := s.Instances()
		instanceHandles := make([]instanceHandle, len(instances))
		for j, inst := range instances {
			instanceIdleSeconds := 0
			if inst.Idle() {
				instanceIdleSeconds = int(nowFn().Sub(inst.IdleTime()).Seconds())
			}

			awayMsg, _ := inst.AwayMessage()
			instanceHandles[j] = instanceHandle{
				Num:         int(inst.Num()),
				IdleSeconds: instanceIdleSeconds,
				IsAway:      inst.Away(),
				AwayMessage: awayMsg,
				IsInvisible: inst.Invisible(),
			}
			ra := inst.RemoteAddr()
			if ra != nil {
				instanceHandles[j].RemoteAddr = ra.Addr().String()
				instanceHandles[j].RemotePort = int(ra.Port())
			}
		}

		var sessionIdleSeconds int
		if s.Idle() {
			sessionIdleSeconds = int(nowFn().Sub(s.IdleTime()).Seconds())
		}

		var awayMessage string
		allAway := s.Away()
		if allAway {
			awayMessage = s.AwayMessage()
		}

		ou.Sessions[i] = sessionHandle{
			ID:            s.IdentScreenName().String(),
			ScreenName:    s.DisplayScreenName().String(),
			OnlineSeconds: int(nowFn().Sub(s.SignonTime()).Seconds()),
			IsAway:        allAway,
			AwayMessage:   awayMessage,
			IdleSeconds:   sessionIdleSeconds,
			IsInvisible:   s.Invisible(),
			IsICQ:         s.UIN() > 0,
			InstanceCount: s.InstanceCount(),
			Instances:     instanceHandles,
		}
	}

	if err := json.NewEncoder(w).Encode(ou); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// deleteSessionHandler handles DELETE /session/{screenname}
func deleteSessionHandler(w http.ResponseWriter, r *http.Request, sessionRetriever SessionRetriever) {
	w.Header().Set("Content-Type", "application/json")
	if screenName := r.PathValue("screenname"); screenName != "" {
		session := sessionRetriever.RetrieveSession(state.NewIdentScreenName(screenName))
		if session == nil {
			errorMsg(w, "session not found", http.StatusNotFound)
			return
		}
		session.CloseSession()
	}
	w.WriteHeader(http.StatusNoContent)
}

// postInstantMessageHandler handles the POST /instant-message endpoint.
func postInstantMessageHandler(w http.ResponseWriter, r *http.Request, messageRelayer MessageRelayer, logger *slog.Logger) {
	input := instantMessage{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "malformed input", http.StatusBadRequest)
		return
	}

	tlv, err := wire.ICBMFragmentList(input.Text)
	if err != nil {
		logger.Error("error sending message POST /instant-message", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	msg := wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.ICBM,
			SubGroup:  wire.ICBMChannelMsgToClient,
		},
		Body: wire.SNAC_0x04_0x07_ICBMChannelMsgToClient{
			ChannelID: 1,
			TLVUserInfo: wire.TLVUserInfo{
				ScreenName: input.From,
			},
			TLVRestBlock: wire.TLVRestBlock{
				TLVList: wire.TLVList{
					wire.NewTLVBE(wire.ICBMTLVAOLIMData, tlv),
				},
			},
		},
	}
	messageRelayer.RelayToScreenName(context.Background(), state.NewIdentScreenName(input.To), msg)
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "Message sent successfully.")
}

// getVersionHandler handles the GET /version endpoint.
func getVersionHandler(w http.ResponseWriter, bld config.Build) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bld); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// postDirectoryCategoryHandler handles the POST /directory/category endpoint.
func postDirectoryCategoryHandler(w http.ResponseWriter, r *http.Request, manager DirectoryManager, logger *slog.Logger) {
	input := directoryCategoryCreate{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errorMsg(w, "malformed input", http.StatusBadRequest)
		return
	}

	category, err := manager.CreateCategory(r.Context(), input.Name)
	if err != nil {
		if errors.Is(err, state.ErrKeywordCategoryExists) {
			errorMsg(w, "category already exists", http.StatusConflict)
		} else {
			logger.Error("error in POST /directory/category", "err", err.Error())
			errorMsg(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	dc := directoryCategory{
		ID:   category.ID,
		Name: category.Name,
	}
	if err := json.NewEncoder(w).Encode(dc); err != nil {
		errorMsg(w, err.Error(), http.StatusBadRequest)
	}
}

// getDirectoryCategoryHandler handles the GET /directory/category endpoint.
func getDirectoryCategoryHandler(w http.ResponseWriter, r *http.Request, manager DirectoryManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	categories, err := manager.Categories(r.Context())
	if err != nil {
		logger.Error("error in GET /directory/category", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	out := make([]directoryCategory, len(categories))
	for i, category := range categories {
		out[i] = directoryCategory{
			ID:   category.ID,
			Name: category.Name,
		}
	}

	if err := json.NewEncoder(w).Encode(out); err != nil {
		errorMsg(w, err.Error(), http.StatusInternalServerError)
	}
}

// getDirectoryCategoryKeywordHandler handles the GET /directory/category/{id}/keyword endpoint.
func getDirectoryCategoryKeywordHandler(w http.ResponseWriter, r *http.Request, manager DirectoryManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	categoryID, err := strconv.ParseUint(r.PathValue("id"), 10, 8)
	if err != nil {
		errorMsg(w, "invalid category ID", http.StatusBadRequest)
		return
	}

	categories, err := manager.KeywordsByCategory(r.Context(), uint8(categoryID))
	if err != nil {
		if errors.Is(err, state.ErrKeywordCategoryNotFound) {
			errorMsg(w, "category not found", http.StatusNotFound)
		} else {
			logger.Error("error in GET /directory/category/{id}/keyword", "err", err.Error())
			errorMsg(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	out := make([]directoryCategory, len(categories))
	for i, category := range categories {
		out[i] = directoryCategory{
			ID:   category.ID,
			Name: category.Name,
		}
	}

	if err := json.NewEncoder(w).Encode(out); err != nil {
		errorMsg(w, err.Error(), http.StatusInternalServerError)
	}
}

// deleteDirectoryCategoryHandler handles the DELETE /directory/category/{id} endpoint.
func deleteDirectoryCategoryHandler(w http.ResponseWriter, r *http.Request, manager DirectoryManager, logger *slog.Logger) {
	categoryID, err := strconv.ParseUint(r.PathValue("id"), 10, 8)
	if err != nil {
		http.Error(w, "invalid category ID", http.StatusBadRequest)
		return
	}

	if err := manager.DeleteCategory(r.Context(), uint8(categoryID)); err != nil {
		switch err {
		case state.ErrKeywordCategoryNotFound:
			errorMsg(w, "category not found", http.StatusNotFound)
			return
		case state.ErrKeywordInUse:
			errorMsg(w, "can't delete because category in use by a user", http.StatusConflict)
			return
		default:
			logger.Error("error in DELETE /directory/category/{id}", "err", err.Error())
			errorMsg(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// postDirectoryKeywordHandler handles the POST /directory/keyword endpoint.
func postDirectoryKeywordHandler(w http.ResponseWriter, r *http.Request, manager DirectoryManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	input := directoryKeywordCreate{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errorMsg(w, "malformed input", http.StatusBadRequest)
		return
	}

	kw, err := manager.CreateKeyword(r.Context(), input.Name, input.CategoryID)
	switch err {
	case state.ErrKeywordCategoryNotFound:
		errorMsg(w, "category not found", http.StatusNotFound)
		return
	case state.ErrKeywordExists:
		errorMsg(w, "keyword already exists", http.StatusConflict)
		return
	case nil:
		w.WriteHeader(http.StatusCreated)
		dc := directoryKeyword{
			ID:   kw.ID,
			Name: kw.Name,
		}
		if err := json.NewEncoder(w).Encode(dc); err != nil {
			errorMsg(w, err.Error(), http.StatusBadRequest)
		}
		return
	default:
		logger.Error("error in POST /directory/keyword", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// deleteDirectoryKeywordHandler handles the DELETE /directory/keyword/{id} endpoint.
func deleteDirectoryKeywordHandler(w http.ResponseWriter, r *http.Request, manager DirectoryManager, logger *slog.Logger) {
	keywordID, err := strconv.ParseUint(r.PathValue("id"), 10, 8)
	if err != nil {
		errorMsg(w, "invalid keyword ID", http.StatusBadRequest)
		return
	}

	if err := manager.DeleteKeyword(r.Context(), uint8(keywordID)); err != nil {
		switch err {
		case state.ErrKeywordInUse:
			errorMsg(w, "can't delete because category in use by a user", http.StatusConflict)
			return
		case state.ErrKeywordNotFound:
			errorMsg(w, "keyword not found", http.StatusNotFound)
			return
		default:
			logger.Error("error in DELETE /directory/keyword/{id}", "err", err.Error())
			errorMsg(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// postPublicChatHandler handles the POST /chat/room/public endpoint.
func postPublicChatHandler(w http.ResponseWriter, r *http.Request, chatRoomCreator ChatRoomCreator, logger *slog.Logger) {
	input := chatRoomCreate{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 50 {
		http.Error(w, "chat room name must be between 1 and 50 characters", http.StatusBadRequest)
		return
	}

	cr := state.NewChatRoom(input.Name, state.NewIdentScreenName("system"), state.PublicExchange)
	err := chatRoomCreator.CreateChatRoom(r.Context(), &cr)
	switch err {
	case state.ErrDupChatRoom:
		http.Error(w, "Chat room already exists.", http.StatusConflict)
		return
	default:
		if err != nil {
			logger.Error("error inserting chat room POST /chat/room/public", "err", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintln(w, "Chat room created successfully.")
}

// getPublicChatHandler handles the GET /chat/room/public endpoint.
func getPublicChatHandler(w http.ResponseWriter, r *http.Request, chatRoomRetriever ChatRoomRetriever, chatSessionRetriever ChatSessionRetriever, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	rooms, err := chatRoomRetriever.AllChatRooms(r.Context(), state.PublicExchange)
	if err != nil {
		logger.Error("error in GET /chat/rooms/public", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	out := make([]chatRoom, len(rooms))
	for i, room := range rooms {
		sessions := chatSessionRetriever.AllSessions(room.Cookie())
		cr := chatRoom{
			CreateTime:   room.CreateTime(),
			Name:         room.Name(),
			Participants: make([]aimChatUserHandle, len(sessions)),
			URL:          room.URL().String(),
		}
		for j, sess := range sessions {
			cr.Participants[j] = aimChatUserHandle{
				ID:         sess.IdentScreenName().String(),
				ScreenName: sess.DisplayScreenName().String(),
			}
		}

		out[i] = cr
	}

	writeUnescapeChatURL(w, out)
}

// deletePublicChatHandler handles the DELETE /chat/room/public endpoint.41
func deletePublicChatHandler(w http.ResponseWriter, r *http.Request, chatRoomDeleter ChatRoomDeleter, logger *slog.Logger) {
	input := chatRoomDelete{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "malformed input", http.StatusBadRequest)
		return
	}

	if len(input.Names) == 0 {
		http.Error(w, "no chat room names provided", http.StatusBadRequest)
		return
	}

	err := chatRoomDeleter.DeleteChatRooms(r.Context(), state.PublicExchange, input.Names)
	if err != nil {
		logger.Error("error deleting public chat rooms DELETE /chat/room/public", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, _ = fmt.Fprintln(w, "Chat rooms deleted successfully.")
}

// getPrivateChatHandler handles the GET /chat/room/private endpoint.
func getPrivateChatHandler(w http.ResponseWriter, r *http.Request, chatRoomRetriever ChatRoomRetriever, chatSessionRetriever ChatSessionRetriever, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	rooms, err := chatRoomRetriever.AllChatRooms(r.Context(), state.PrivateExchange)
	if err != nil {
		logger.Error("error in GET /chat/rooms/private", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	out := make([]chatRoom, len(rooms))
	for i, room := range rooms {
		sessions := chatSessionRetriever.AllSessions(room.Cookie())
		cr := chatRoom{
			CreateTime:   room.CreateTime(),
			CreatorID:    room.Creator().String(),
			Name:         room.Name(),
			Participants: make([]aimChatUserHandle, len(sessions)),
			URL:          room.URL().String(),
		}
		for j, sess := range sessions {
			cr.Participants[j] = aimChatUserHandle{
				ID:         sess.IdentScreenName().String(),
				ScreenName: sess.DisplayScreenName().String(),
			}
		}

		out[i] = cr
	}

	writeUnescapeChatURL(w, out)
}

// getFeedbagBuddyHandler handles the GET /feedbag/{screen_name}/group endpoint.
func getFeedbagBuddyHandler(w http.ResponseWriter, r *http.Request, feedbagManager FeedbagManager, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	screenName := r.PathValue("screen_name")
	if screenName == "" {
		errorMsg(w, "screen_name is required", http.StatusBadRequest)
		return
	}

	items, err := feedbagManager.Feedbag(r.Context(), state.NewIdentScreenName(screenName))
	if err != nil {
		logger.Error("error retrieving feedbag", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	} else if len(items) == 0 {
		errorMsg(w, "feedbag not found", http.StatusNotFound)
		return
	}

	buddyMap := make(map[uint16][]*wire.FeedbagItem)
	for _, item := range items {
		switch item.ClassID {
		case wire.FeedbagClassIdBuddy:
			buddyMap[item.GroupID] = append(buddyMap[item.GroupID], &item)
		}
	}

	type buddyItem struct {
		Name   string `json:"name"`
		ItemID uint16 `json:"item_id"`
	}

	type groupItem struct {
		GroupID   uint16      `json:"group_id"`
		GroupName string      `json:"group_name"`
		Buddies   []buddyItem `json:"buddies"`
	}

	response := make([]groupItem, 0)
	for _, item := range items {
		switch item.ClassID {
		case wire.FeedbagClassIdGroup:
			if item.GroupID == 0 {
				// can't add buddies to the root group
				continue
			}

			group := groupItem{
				GroupID:   item.GroupID,
				GroupName: item.Name,
				Buddies:   make([]buddyItem, 0, len(buddyMap[item.GroupID])),
			}
			for _, buddy := range buddyMap[item.GroupID] {
				group.Buddies = append(group.Buddies, buddyItem{
					Name:   buddy.Name,
					ItemID: buddy.ItemID,
				})
			}

			response = append(response, group)
		}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("error encoding response", "err", err.Error())
	}
}

// putFeedbagBuddyHandler handles the PUT /feedbag/{screen_name}/group/{group_id}/buddy/{buddy_screen_name} endpoint.
func putFeedbagBuddyHandler(w http.ResponseWriter, r *http.Request, buddyBroadcaster BuddyBroadcaster, feedbagManager FeedbagManager, sessionRetriever SessionRetriever, messageRelayer MessageRelayer, logger *slog.Logger, randInt func(n int) int) {
	w.Header().Set("Content-Type", "application/json")
	gid, err := strconv.ParseUint(r.PathValue("group_id"), 10, 16)
	if err != nil {
		errorMsg(w, "invalid group_id", http.StatusBadRequest)
		return
	}
	groupID := uint16(gid)

	if groupID == 0 {
		errorMsg(w, "can't add buddies to root group", http.StatusBadRequest)
		return
	}

	screenName := r.PathValue("screen_name")
	if screenName == "" {
		errorMsg(w, "screen_name is required", http.StatusBadRequest)
		return
	}
	me := state.NewIdentScreenName(screenName)

	buddyScreenName := r.PathValue("buddy_screen_name")
	if buddyScreenName == "" {
		errorMsg(w, "buddy_screen_name is required", http.StatusBadRequest)
		return
	}

	newBuddy := state.DisplayScreenName(buddyScreenName)
	if newBuddy.IsUIN() {
		if err := newBuddy.ValidateUIN(); err != nil {
			errorMsg(w, fmt.Sprintf("invalid uin: %s", err), http.StatusBadRequest)
			return
		}
	} else {
		if err := newBuddy.ValidateAIMHandle(); err != nil {
			errorMsg(w, fmt.Sprintf("invalid screen name: %s", err), http.StatusBadRequest)
			return
		}
	}

	items, err := feedbagManager.Feedbag(r.Context(), me)
	if err != nil {
		logger.Error("error retrieving feedbag", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var count int
	var group *wire.FeedbagItem
	for _, item := range items {
		switch {
		case item.ClassID == wire.FeedbagClassIdGroup && item.GroupID == groupID:
			group = &item
		case item.ClassID == wire.FeedbagClassIdBuddy && item.GroupID == groupID:
			count++
			if item.Name == newBuddy.IdentScreenName().String() {
				response := struct {
					Name    string `json:"name"`
					GroupID uint16 `json:"group_id"`
					ItemID  uint16 `json:"item_id"`
				}{
					Name:    buddyScreenName,
					GroupID: groupID,
					ItemID:  item.ItemID,
				}
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(response); err != nil {
					logger.Error("error encoding response", "err", err.Error())
				}
				return
			}
		}
	}

	if count >= 30 {
		errorMsg(w, "too many buddies in group. max: 30", http.StatusBadRequest)
		return
	}

	if group == nil {
		errorMsg(w, "group not found", http.StatusNotFound)
		return
	}

	buddyItem := wire.FeedbagItem{
		Name:    buddyScreenName,
		GroupID: groupID,
		ItemID:  randItemID(randInt, items),
		ClassID: wire.FeedbagClassIdBuddy,
	}
	if buddyItem.ItemID == 0 {
		errorMsg(w, "maximum items reached", http.StatusConflict)
		return
	}

	if order, hasOrder := group.Bytes(wire.FeedbagAttributesOrder); hasOrder {
		var memberIDs []uint16
		if err := wire.UnmarshalBE(&memberIDs, bytes.NewReader(order)); err != nil {
			logger.Error("error decoding order TLV", "err", err.Error())
			errorMsg(w, "internal server error", http.StatusInternalServerError)
			return
		}
		group.Replace(wire.NewTLVBE(wire.FeedbagAttributesOrder, append(memberIDs, buddyItem.ItemID)))
	} else {
		group.Append(wire.NewTLVBE(wire.FeedbagAttributesOrder, []uint16{buddyItem.ItemID}))
	}

	updates := []wire.FeedbagItem{
		buddyItem,
		*group,
	}
	if err := feedbagManager.FeedbagUpsert(r.Context(), me, updates); err != nil {
		logger.Error("error inserting feedbag item", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session := sessionRetriever.RetrieveSession(me)
	if session != nil {
		messageRelayer.RelayToScreenName(r.Context(), me, wire.SNACMessage{
			Frame: wire.SNACFrame{
				FoodGroup: wire.Feedbag,
				SubGroup:  wire.FeedbagInsertItem,
				RequestID: wire.ReqIDFromServer,
			},
			Body: wire.SNAC_0x13_0x09_FeedbagUpdateItem{
				Items: []wire.FeedbagItem{buddyItem},
			},
		})
		messageRelayer.RelayToScreenName(r.Context(), me, wire.SNACMessage{
			Frame: wire.SNACFrame{
				FoodGroup: wire.Feedbag,
				SubGroup:  wire.FeedbagUpdateItem,
				RequestID: wire.ReqIDFromServer,
			},
			Body: wire.SNAC_0x13_0x09_FeedbagUpdateItem{
				Items: []wire.FeedbagItem{*group},
			},
		})
		instances := session.Instances()
		if len(instances) > 0 {
			if err := buddyBroadcaster.BroadcastVisibility(r.Context(), instances[0], []state.IdentScreenName{newBuddy.IdentScreenName()}, false); err != nil {
				logger.Error("error broadcasting visibility", "err", err.Error())
			}
		}
	}

	response := struct {
		Name    string `json:"name"`
		GroupID uint16 `json:"group_id"`
		ItemID  uint16 `json:"item_id"`
	}{
		Name:    buddyItem.Name,
		GroupID: buddyItem.GroupID,
		ItemID:  buddyItem.ItemID,
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("error encoding response", "err", err.Error())
	}
}

// deleteFeedbagBuddyHandler handles the DELETE /feedbag/{screen_name}/group/{group_id}/buddy/{buddy_screen_name} endpoint.
func deleteFeedbagBuddyHandler(w http.ResponseWriter, r *http.Request, buddyBroadcaster BuddyBroadcaster, feedbagManager FeedbagManager, sessionRetriever SessionRetriever, messageRelayer MessageRelayer, logger *slog.Logger) {
	gid, err := strconv.ParseUint(r.PathValue("group_id"), 10, 16)
	if err != nil {
		errorMsg(w, "invalid group_id", http.StatusBadRequest)
		return
	}

	groupID := uint16(gid)
	if groupID == 0 {
		errorMsg(w, "can't add buddies to root group", http.StatusBadRequest)
		return
	}

	screenName := r.PathValue("screen_name")
	if screenName == "" {
		errorMsg(w, "screen_name is required", http.StatusBadRequest)
		return
	}

	me := state.NewIdentScreenName(screenName)
	buddyScreenName := r.PathValue("buddy_screen_name")
	if buddyScreenName == "" {
		errorMsg(w, "buddy_screen_name is required", http.StatusBadRequest)
		return
	}

	deleteBuddy := state.NewIdentScreenName(buddyScreenName)
	items, err := feedbagManager.Feedbag(r.Context(), me)
	if err != nil {
		logger.Error("error retrieving feedbag", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var groupFound bool
	var itemToDelete *wire.FeedbagItem
	for _, item := range items {
		switch {
		case item.ClassID == wire.FeedbagClassIdGroup && item.GroupID == groupID:
			groupFound = true
		case item.ClassID == wire.FeedbagClassIdBuddy && item.Name == buddyScreenName && item.GroupID == groupID:
			itemToDelete = &item
		}
	}

	switch {
	case !groupFound:
		errorMsg(w, "group not found", http.StatusNotFound)
		return
	case itemToDelete == nil:
		errorMsg(w, "buddy not found", http.StatusNotFound)
		return
	}

	if err := feedbagManager.FeedbagDelete(r.Context(), me, []wire.FeedbagItem{*itemToDelete}); err != nil {
		logger.Error("error deleting feedbag item", "err", err.Error())
		errorMsg(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session := sessionRetriever.RetrieveSession(me)
	if session != nil {
		messageRelayer.RelayToScreenName(r.Context(), me, wire.SNACMessage{
			Frame: wire.SNACFrame{
				FoodGroup: wire.Feedbag,
				SubGroup:  wire.FeedbagDeleteItem,
				RequestID: wire.ReqIDFromServer,
			},
			Body: wire.SNAC_0x13_0x0A_FeedbagDeleteItem{
				Items: []wire.FeedbagItem{*itemToDelete},
			},
		})
		instances := session.Instances()
		if len(instances) > 0 {
			if err := buddyBroadcaster.BroadcastVisibility(r.Context(), instances[0], []state.IdentScreenName{deleteBuddy}, true); err != nil {
				logger.Error("error broadcasting visibility", "err", err.Error())
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
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

// errorMsg sends an error response message and code.
func errorMsg(w http.ResponseWriter, error string, code int) {
	msg := messageBody{Message: error}
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeUnescapeChatURL writes a JSON-encoded list of
// chat rooms with unescaped ampersands preceding the exchange query param.
//
//	before: aim:gochat?roomname=Office+Hijinks\u0026exchange=5
//	after:  aim:gochat?roomname=Office+Hijinks&exchange=5
//
// This makes it easier to copy the gochat URL into AIM,
// which does not recognize the ampersand unicode character \u0026.
func writeUnescapeChatURL(w http.ResponseWriter, out []chatRoom) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(out); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := bytes.ReplaceAll(buf.Bytes(), []byte(`\u0026exchange`), []byte(`&exchange`))
	if _, err := w.Write(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func randItemID(randInt func(n int) int, items []wire.FeedbagItem) uint16 {
	num := uint16(randInt(math.MaxUint16))
	for itemID := num; itemID != num-1; itemID++ {
		if itemID == 0 {
			continue
		}

		var exists bool
		for _, item := range items {
			if item.GroupID == itemID || item.ItemID == itemID {
				exists = true
				break
			}
		}

		if !exists {
			return itemID
		}
	}

	return 0
}
