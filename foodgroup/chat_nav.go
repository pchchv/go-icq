package foodgroup

import "log/slog"

// ChatNavService provides functionality for the ChatNav food group,
// which handles chat room creation and serving chat room metadata.
type ChatNavService struct {
	logger          *slog.Logger
	chatRoomManager ChatRoomRegistry
}

// NewChatNavService creates a new instance of NewChatNavService.
func NewChatNavService(logger *slog.Logger, chatRoomManager ChatRoomRegistry) *ChatNavService {
	return &ChatNavService{
		logger:          logger,
		chatRoomManager: chatRoomManager,
	}
}
