package foodgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pchchv/go-icq/wire"
)

// ODirService provides functionality for the ODir food group,
// which provides functionality for searching the user directory.
type ODirService struct {
	logger         *slog.Logger
	profileManager ProfileManager
}

// NewODirService creates a new instance of ODirService.
func NewODirService(logger *slog.Logger, profileManager ProfileManager) ODirService {
	return ODirService{
		logger:         logger,
		profileManager: profileManager,
	}
}

// KeywordListQuery returns a list of keywords that can be searched in the user directory.
func (s ODirService) KeywordListQuery(ctx context.Context, inFrame wire.SNACFrame) (wire.SNACMessage, error) {
	interests, err := s.profileManager.InterestList(ctx)
	if err != nil {
		return wire.SNACMessage{}, fmt.Errorf("InterestList: %w", err)
	}

	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.ODir,
			SubGroup:  wire.ODirKeywordListReply,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x0F_0x04_KeywordListReply{
			Status:    0x01,
			Interests: interests,
		},
	}, nil
}
