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

// convertTimestampsInMap recursively converts int64 values that look like timestamps to float64.
func convertTimestampsInMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, val := range data {
		// check if key suggests it's a timestamp
		if isTimestampField(key) {
			if intVal, ok := val.(int64); ok {
				result[key] = float64(intVal)
				continue
			}
		}

		// recursively process nested maps
		if nestedMap, ok := val.(map[string]interface{}); ok {
			result[key] = convertTimestampsInMap(nestedMap)
		} else if nestedSlice, ok := val.([]interface{}); ok {
			convertedSlice := make([]interface{}, len(nestedSlice))
			for i, item := range nestedSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					convertedSlice[i] = convertTimestampsInMap(itemMap)
				} else {
					convertedSlice[i] = item
				}
			}
			result[key] = convertedSlice
		} else {
			result[key] = val
		}
	}
	return result
}
