package webapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pchchv/go-icq/server/webapi/handlers"
	"github.com/pchchv/go-icq/server/webapi/middleware"
	"github.com/pchchv/go-icq/state"
	"golang.org/x/sync/errgroup"
)

// Server hosts an HTTP endpoint capable of handling AIM-style Kerberos authentication.
// The messages are structured as SNACs transmitted over HTTP.
type Server struct {
	servers []*http.Server
	logger  *slog.Logger
}

func NewServer(listeners []string, logger *slog.Logger, handler Handler, apiKeyValidator middleware.APIKeyValidator, sessionManager *state.WebAPISessionManager) *Server {
	servers := make([]*http.Server, 0, len(listeners))
	// create authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(apiKeyValidator, logger)
	// create handlers
	authHandler := &handlers.AuthHandler{
		UserManager: handler.UserManager,
		TokenStore:  handler.TokenStore,
		Logger:      logger,
		DisableAuth: handler.OSCARConfig.IsAuthDisabled(),
	}
	sessionHandler := &handlers.SessionHandler{
		SessionManager:      sessionManager,
		OSCARSessionManager: handler.SessionRetriever.(handlers.SessionManager),
		OSCARAuthService:    handler.AuthService,
		BuddyListService:    nil,
		BuddyListRegistry:   handler.BuddyListRegistry,
		BuddyBroadcaster:    handler.BuddyBroadcaster,
		BuddyListManager:    handler.BuddyListManager.(*handlers.BuddyListManager),
		TokenStore:          handler.TokenStore,
		Logger:              logger,
	}
	eventsHandler := &handlers.EventsHandler{
		SessionManager: sessionManager,
		Logger:         logger,
	}
	presenceHandler := &handlers.PresenceHandler{
		SessionManager:      sessionManager,
		SessionRetriever:    handler.SessionRetriever,
		FeedbagRetriever:    handler.FeedbagRetriever,
		BuddyBroadcaster:    handler.BuddyBroadcaster,
		ProfileManager:      handler.ProfileManager,
		RelationshipFetcher: handler.RelationshipFetcher,
		Logger:              logger,
	}
	buddyListHandler := &handlers.BuddyListHandler{
		SessionManager: sessionManager,
		FeedbagManager: handler.FeedbagManager,
		Logger:         logger,
	}
	// messaging handler
	messagingHandler := &handlers.MessagingHandler{
		SessionManager:        sessionManager,
		MessageRelayer:        handler.MessageRelayer,
		OfflineMessageManager: handler.OfflineMessageManager,
		SessionRetriever:      handler.SessionRetriever,
		RelationshipFetcher:   handler.RelationshipFetcher,
		Logger:                logger,
	}
	// preference handler
	preferenceHandler := &handlers.PreferenceHandler{
		SessionManager:    sessionManager,
		PreferenceManager: handler.PreferenceManager,
		PermitDenyManager: handler.PermitDenyManager,
		Logger:            logger,
	}
	// OSCAR Bridge handler
	oscarBridgeHandler := &handlers.OSCARBridgeHandler{
		SessionManager:   sessionManager,
		OSCARAuthService: handler.AuthService,
		CookieBaker:      handler.CookieBaker,
		BridgeStore:      handler.OSCARBridgeStore,
		Config:           handler.OSCARConfig,
		Logger:           logger,
	}
	// chat handler
	chatHandler := &handlers.ChatHandler{
		SessionManager: sessionManager,
		ChatManager:    handler.ChatManager,
		Logger:         logger,
	}
	for _, l := range listeners {
		mux := http.NewServeMux()
		// public endpoint (no auth required for hello world)
		mux.HandleFunc("GET /", handler.GetHelloWorldHandler)
		// authentication endpoint
		// (public - no API key required for user login)
		mux.HandleFunc("POST /auth/clientLogin", func(w http.ResponseWriter, r *http.Request) {
			// set CORS headers for public endpoint
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			authHandler.ClientLogin(w, r)
		})
		// handle OPTIONS for CORS preflight
		mux.HandleFunc("OPTIONS /auth/clientLogin", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
		})
		// authenticated Web AIM API endpoints
		// SessionInstance management
		// supports multiple auth methods (k, a, ts+sig_sha256)
		mux.Handle("GET /aim/startSession", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(sessionHandler.StartSession))))
		// end session
		// uses aimsid for auth, no k required
		mux.Handle("GET /aim/endSession", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(sessionHandler.EndSession))))
		// event fetching
		// uses aimsid for auth, no k required
		mux.Handle("GET /aim/fetchEvents", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(eventsHandler.FetchEvents))))
		// add temp buddy
		// uses aimsid for auth
		mux.Handle("GET /aim/addTempBuddy", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(buddyListHandler.AddTempBuddy))))
		// presence and buddy list
		// GetPresence supports aimsid-based auth, so we use flexible auth
		mux.Handle("GET /presence/get", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(presenceHandler.GetPresence))))
		mux.Handle("GET /buddylist/addBuddy", authMiddleware.Authenticate(authMiddleware.CORSMiddleware(http.HandlerFunc(buddyListHandler.AddBuddy))))
		// messaging endpoints
		// sendIM supports aimsid-based auth, so we use flexible auth
		mux.Handle("GET /im/sendIM", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(messagingHandler.SendIM))))
		mux.Handle("GET /im/setTyping", authMiddleware.Authenticate(authMiddleware.CORSMiddleware(http.HandlerFunc(messagingHandler.SetTyping))))
		// presence management endpoints
		// SetState only requires aimsid, no k parameter needed
		mux.Handle("GET /presence/setState", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(presenceHandler.SetState))))
		// these presence endpoints support aimsid-based auth where k is not required
		mux.Handle("GET /presence/setStatus", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(presenceHandler.SetStatus))))
		mux.Handle("GET /presence/setProfile", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(presenceHandler.SetProfile))))
		mux.Handle("GET /presence/getProfile", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(presenceHandler.GetProfile))))
		// presence icon endpoint (no auth required)
		mux.HandleFunc("GET /presence/icon", presenceHandler.Icon)
		// preference management endpoints
		// these endpoints support aimsid-based auth, so we use a flexible auth approach
		mux.Handle("GET /preference/set", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(preferenceHandler.SetPreferences))))
		mux.Handle("GET /preference/get", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(preferenceHandler.GetPreferences))))
		mux.Handle("GET /preference/setPermitDeny", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(preferenceHandler.SetPermitDeny))))
		mux.Handle("GET /preference/getPermitDeny", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(preferenceHandler.GetPermitDeny))))
		// advanced features
		// OSCAR Bridge endpoint
		mux.Handle("GET /aim/startOSCARSession", authMiddleware.Authenticate(authMiddleware.CORSMiddleware(http.HandlerFunc(oscarBridgeHandler.StartOSCARSession))))
		// expressions endpoint (for buddy icons, etc.)
		expressionsHandler := handlers.NewExpressionsHandler(logger)
		mux.Handle("GET /expressions/get", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(expressionsHandler.Get))))
		// chat room endpoints
		// all chat endpoints use aimsid for authentication
		mux.Handle("GET /chat/createAndJoinChat", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(chatHandler.CreateAndJoinChat))))
		mux.Handle("GET /chat/sendMessage", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(chatHandler.SendMessage))))
		mux.Handle("GET /chat/setTyping", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(chatHandler.SetTyping))))
		mux.Handle("GET /chat/leaveChat", authMiddleware.AuthenticateFlexible(authMiddleware.CORSMiddleware(http.HandlerFunc(chatHandler.LeaveChat))))
		servers = append(servers, &http.Server{
			Addr:    l,
			Handler: mux,
		})
	}

	return &Server{
		servers: servers,
		logger:  logger,
	}
}

func (s *Server) ListenAndServe() error {
	if len(s.servers) == 0 {
		s.logger.Debug("no webapi listeners defined")
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
				return fmt.Errorf("unable to start webapi server: %w", err)
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
