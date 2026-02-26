package foodgroup

import (
	"fmt"
	"log/slog"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

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

// sendChatNavErrorSNAC returns a ChatNavErr SNAC and logs an error for the operator.
func sendChatNavErrorSNAC(inFrame wire.SNACFrame, errorCode uint16) (wire.SNACMessage, error) {
	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.ChatNav,
			SubGroup:  wire.ChatNavErr,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNACError{
			Code: errorCode,
		},
	}, nil
}

func validateExchange(exchange uint16) error {
	if exchange == state.PrivateExchange || exchange == state.PublicExchange {
		return nil
	}
	return fmt.Errorf("only exchanges %d and %d are supported", state.PrivateExchange, state.PublicExchange)
}
