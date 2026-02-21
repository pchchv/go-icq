package foodgroup

// UserLookupService implements the UserLookup food group.
type UserLookupService struct {
	profileManager ProfileManager
}

// NewUserLookupService returns a new instance of UserLookupService.
func NewUserLookupService(profileManager ProfileManager) UserLookupService {
	return UserLookupService{
		profileManager: profileManager,
	}
}
