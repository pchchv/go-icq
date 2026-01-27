package handlers

import "github.com/pchchv/go-icq/server/webapi/types"

// ConvertEventForAMF3 converts a WebAPIEvent to a map suitable for AMF3 encoding,
// ensuring all timestamps are float64 to avoid uint29 overflow issues.
func ConvertEventForAMF3(event types.Event) map[string]interface{} {
	result := map[string]interface{}{
		"type":      string(event.Type),
		"seqNum":    event.SeqNum,
		"timestamp": float64(event.Timestamp), // convert to float64
	}

	// convert event data based on type
	switch event.Type {
	case types.EventTypeIM:
		if imEvent, ok := event.Data.(types.IMEvent); ok {
			// gromit expects 'source' as a user object and 'autoresponse' (lowercase)
			result["eventData"] = map[string]interface{}{
				"source": map[string]interface{}{
					"aimId": imEvent.From,
				},
				"message":      imEvent.Message,
				"timestamp":    imEvent.Timestamp, // Already float64
				"autoresponse": imEvent.AutoResp,
			}
		} else if dataMap, ok := event.Data.(map[string]interface{}); ok {
			// already a map, ensure timestamps are float64
			if ts, exists := dataMap["timestamp"]; exists {
				if tsInt, ok := ts.(int64); ok {
					dataMap["timestamp"] = float64(tsInt)
				}
			}
			result["eventData"] = dataMap
		} else {
			result["eventData"] = event.Data
		}
	case types.EventTypeOfflineIM:
		if imEvent, ok := event.Data.(types.IMEvent); ok {
			result["eventData"] = map[string]interface{}{
				"aimId":     imEvent.From,
				"message":   imEvent.Message,
				"timestamp": float64(imEvent.Timestamp), // Convert to float64
			}
		} else if dataMap, ok := event.Data.(map[string]interface{}); ok {
			// already a map, ensure timestamps are float64
			if ts, exists := dataMap["timestamp"]; exists {
				if tsInt, ok := ts.(int64); ok {
					dataMap["timestamp"] = float64(tsInt)
				}
			}
			result["eventData"] = dataMap
		} else {
			result["eventData"] = event.Data
		}
	case types.EventTypePresence:
		if presenceEvent, ok := event.Data.(types.PresenceEvent); ok {
			eventData := map[string]interface{}{
				"aimId":    presenceEvent.AimID,
				"state":    presenceEvent.State,
				"userType": presenceEvent.UserType,
			}
			// convert timestamp fields to float64
			if presenceEvent.OnlineTime > 0 {
				eventData["onlineTime"] = float64(presenceEvent.OnlineTime)
			}
			result["eventData"] = eventData
		} else if dataMap, ok := event.Data.(map[string]interface{}); ok {
			// already a map, ensure timestamps are float64
			if ot, exists := dataMap["onlineTime"]; exists {
				if otInt, ok := ot.(int64); ok {
					dataMap["onlineTime"] = float64(otInt)
				}
			}
			result["eventData"] = dataMap
		} else {
			result["eventData"] = event.Data
		}
	case types.EventType("myInfo"):
		// myInfo events often contain timestamps
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			// convert any int64 timestamps to float64
			for key, val := range dataMap {
				if key == "onlineTime" || key == "memberSince" || key == "awayTime" || key == "statusTime" {
					if intVal, ok := val.(int64); ok {
						dataMap[key] = float64(intVal)
					}
				}
			}
			result["eventData"] = dataMap
		} else {
			result["eventData"] = event.Data
		}
	case types.EventTypeBuddyList:
		// buddy list events are already converted to maps in FormatBuddyListEvent
		// Just pass through
		result["eventData"] = event.Data
	case types.EventTypeTyping:
		if typingEvent, ok := event.Data.(types.TypingEvent); ok {
			// gromit expects 'aimId' and 'typingStatus'
			result["eventData"] = map[string]interface{}{
				"aimId":        typingEvent.From,
				"typingStatus": typingEvent.Typing,
			}
		} else {
			result["eventData"] = event.Data
		}
	case types.EventTypeSentIM:
		if sentIMEvent, ok := event.Data.(types.SentIMEvent); ok {
			// gromit expects both 'source' (sender) and 'dest' (recipient) for sentIM
			// the parseIM function needs source even for outgoing messages
			result["eventData"] = map[string]interface{}{
				"source": map[string]interface{}{
					"aimId":     sentIMEvent.Sender.AimID,
					"displayId": sentIMEvent.Sender.DisplayID,
					"userType":  sentIMEvent.Sender.UserType,
					"state":     "online",
				},
				"dest": map[string]interface{}{
					"aimId":     sentIMEvent.Dest.AimID,
					"displayId": sentIMEvent.Dest.DisplayID,
					"userType":  sentIMEvent.Dest.UserType,
					"state":     "online",
				},
				"message":      sentIMEvent.Message,
				"timestamp":    sentIMEvent.Timestamp, // Already float64
				"autoresponse": sentIMEvent.AutoResp,
			}
		} else {
			result["eventData"] = event.Data
		}
	default:
		// unknown types
		// check if data is a map and convert any int64 values
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			result["eventData"] = convertTimestampsInMap(dataMap)
		} else {
			result["eventData"] = event.Data
		}
	}

	return result
}

// ConvertEventsForAMF3 converts a slice of WebAPIEvents for AMF3 encoding.
func ConvertEventsForAMF3(events []types.Event) []interface{} {
	result := make([]interface{}, len(events))
	for i, event := range events {
		result[i] = ConvertEventForAMF3(event)
	}
	return result
}

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
