package state

// Relationship represents the relationship between two users.
// Users A and B are related if:
//   - A has user B on their buddy list, or vice versa
//   - A has user B on their deny list, or vice versa
//   - A has user B on their permit list, or vice versa
type Relationship struct {
	// User is the screen name of the user with whom you have a relationship.
	User IdentScreenName
	// BlocksYou indicates whether user blocks you.
	// This is true when user has the following permit/deny modes set:
	// 	- DenyAll
	// 	- PermitSome (and you are not on permit list)
	// 	- DenySome (and you are on deny list)
	// 	- PermitOnList (and you are not on their buddy list)
	BlocksYou bool
	// YouBlock indicates whether you block user.
	// This is true when user has the following permit/deny modes set:
	// 	- DenyAll
	// 	- PermitSome (and they are not on your permit list)
	// 	- DenySome (and they are on your deny list)
	// 	- PermitOnList (and they are not on your buddy list)
	YouBlock bool
	// IsOnTheirList indicates whether you are on user's buddy list.
	IsOnTheirList bool
	// IsOnYourList indicates whether this user is on your buddy list.
	IsOnYourList bool
}
