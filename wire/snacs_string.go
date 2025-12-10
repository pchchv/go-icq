package wire

var foodGroupName = map[uint16]string{
	OService:    "OService",
	Locate:      "Locate",
	Buddy:       "Buddy",
	ICBM:        "ICBM",
	Advert:      "Advert",
	Invite:      "Invite",
	Admin:       "Admin",
	Popup:       "Popup",
	PermitDeny:  "PermitDeny",
	UserLookup:  "UserLookup",
	Stats:       "Stats",
	Translate:   "Translate",
	ChatNav:     "ChatNav",
	Chat:        "Chat",
	ODir:        "ODir",
	BART:        "BART",
	Feedbag:     "Feedbag",
	ICQ:         "ICQ",
	BUCP:        "BUCP",
	Alert:       "Alert",
	Plugin:      "Plugin",
	UnnamedFG24: "UnnamedFG24",
	MDir:        "MDir",
	ARS:         "ARS",
}

// FoodGroupName gets the string name of a food group.
// It returns "unknown" if the food group doesn't exist.
func FoodGroupName(foodGroup uint16) string {
	name := foodGroupName[foodGroup]
	if name == "" {
		name = "unknown"
	}
	return name
}
