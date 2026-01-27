package handlers

// isTimestampField checks if a field name suggests it contains a timestamp.
func isTimestampField(fieldName string) bool {
	timestampFields := []string{
		"awayTime", "AwayTime",
		"idleTime", "IdleTime",
		"timestamp", "Timestamp",
		"onlineTime", "OnlineTime",
		"memberSince", "MemberSince",
		"statusTime", "StatusTime",
		"loginTime", "LoginTime",
		"createdAt", "CreatedAt",
		"updatedAt", "UpdatedAt",
	}

	for _, tf := range timestampFields {
		if fieldName == tf {
			return true
		}
	}

	return false
}
