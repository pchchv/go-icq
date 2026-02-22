package foodgroup

import "log/slog"

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
