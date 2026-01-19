package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/pchchv/go-icq/server/webapi/types"
)

const AMF3 AMFVersion = 3

// AMFVersion represents the AMF encoding version.
type AMFVersion int

// AMFEncoder handles AMF encoding operations for WebAPI responses.
type AMFEncoder struct {
	logger *slog.Logger
}

// NewAMFEncoder creates a new AMF encoder instance.
func NewAMFEncoder(logger *slog.Logger) *AMFEncoder {
	return &AMFEncoder{logger: logger}
}

// isZeroValue checks if a reflect.Value is a zero value.
func (e *AMFEncoder) isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		// for time.Time, check if it's zero
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
		// for other structs, we can't easily determine zero value
		return false
	default:
		return false
	}
}

// toAMF3Compatible converts Go types to AMF3-compatible format for goAMF3.
func (e *AMFEncoder) toAMF3Compatible(data interface{}) interface{} {
	return map[string]interface{}{}
}

// mapToAMFMap converts a Go map to an AMF3-compatible map.
func (e *AMFEncoder) mapToAMFMap(v reflect.Value) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range v.MapKeys() {
		// convert key to string (AMF only supports string keys)
		keyStr := fmt.Sprintf("%v", key.Interface())
		value := v.MapIndex(key)
		if value.CanInterface() {
			result[keyStr] = e.toAMF3Compatible(value.Interface())
		}
	}
	return result
}

// structToMap converts a struct to a map using JSON tags for AMF3.
func (e *AMFEncoder) structToMap(v reflect.Value) map[string]interface{} {
	result := make(map[string]interface{})
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		// skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// parse JSON tag
		tagParts := strings.Split(jsonTag, ",")
		fieldName := tagParts[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		// check for omitempty
		omitEmpty := false
		for _, part := range tagParts[1:] {
			if part == "omitempty" {
				omitEmpty = true
				break
			}
		}

		// skip if omitempty and value is zero
		if omitEmpty && e.isZeroValue(fieldValue) {
			continue
		}

		// get field value and convert recursively
		fieldData := fieldValue.Interface()
		result[fieldName] = e.toAMF3Compatible(fieldData)
	}

	return result
}

// convertToMap converts any data to a map structure for AMF3.
func (e *AMFEncoder) convertToMap(data interface{}) interface{} {
	if data == nil {
		// for AMF3, return empty map instead of nil to avoid truncation
		return map[string]interface{}{}
	}

	// if already a map, return as-is (even if empty)
	if m, ok := data.(map[string]interface{}); ok {
		if m == nil {
			return map[string]interface{}{}
		}
		return m
	}

	v := reflect.ValueOf(data)
	// handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		data = v.Interface()
	}

	// handle different types
	switch v.Kind() {
	case reflect.Struct:
		return e.structToMap(v)
	case reflect.Map:
		return e.mapToAMFMap(v)
	case reflect.Slice, reflect.Array:
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.CanInterface() {
				result[i] = e.convertToMap(elem.Interface())
			}
		}
		return result
	default:
		// for basic types, return as-is
		return data
	}
}

// responseBodyToMap converts ResponseBody to AMF3-compatible map.
func (e *AMFEncoder) responseBodyToMap(body ResponseBody) map[string]interface{} {
	m := map[string]interface{}{
		"statusCode": body.StatusCode,
		"statusText": body.StatusText,
	}
	if body.Data != nil {
		m["data"] = e.toAMF3Compatible(body.Data)
	} else {
		// for AMF3, always include data field even if empty to prevent truncation
		m["data"] = map[string]interface{}{}
	}
	return m
}

// errorResponseToMap converts ErrorResponse to AMF3-compatible map.
func (e *AMFEncoder) errorResponseToMap(err ErrorResponse) map[string]interface{} {
	return map[string]interface{}{
		"response": map[string]interface{}{
			"statusCode": err.Response.StatusCode,
			"statusText": err.Response.StatusText,
		},
	}
}

// baseResponseToMap converts BaseResponse to AMF3-compatible map.
func (e *AMFEncoder) baseResponseToMap(resp BaseResponse) map[string]interface{} {
	return map[string]interface{}{
		"response": e.responseBodyToMap(resp.Response),
	}
}

// sliceToArray converts a slice to an AMF3-compatible array.
func (e *AMFEncoder) sliceToArray(v reflect.Value) []interface{} {
	length := v.Len()
	result := make([]interface{}, length)
	for i := 0; i < length; i++ {
		elem := v.Index(i)
		if elem.CanInterface() {
			result[i] = e.toAMF3Compatible(elem.Interface())
		} else {
			result[i] = nil
		}
	}
	return result
}

// toAMFCompatible converts Go types to AMF3-compatible types.
func (e *AMFEncoder) toAMFCompatible(data interface{}) interface{} {
	return e.toAMF3Compatible(data)
}

// sanitizeForAMF3 recursively removes nil values from
// the data structure because goAMF3 panics when encountering nil values in maps.
func (e *AMFEncoder) sanitizeForAMF3(data interface{}) interface{} {
	if data == nil {
		return map[string]interface{}{}
	}

	switch v := data.(type) {
	case uint64:
		// goAMF3 can't handle uint64, convert to int
		return int(v)
	case uint32:
		// convert all unsigned to signed for safety
		return int(v)
	case uint16:
		return int(v)
	case uint8:
		return int(v)
	case uint:
		return int(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if val == nil {
				// for fields like 'data', replace with empty map
				// for other fields, skip them
				if key == "data" {
					result[key] = map[string]interface{}{}
				}
				continue
			}
			result[key] = e.sanitizeForAMF3(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = e.sanitizeForAMF3(item)
		}
		return result
	case []types.Event:
		// handle WebAPIEvent arrays specially
		result := make([]interface{}, len(v))
		for i, event := range v {
			// AMF3 has a 29-bit limit for integers
			// keep seqNum small by using modulo
			seqNum := int(event.SeqNum % (1 << 29))
			// convert timestamp to seconds ago to keep it small
			timestampSec := int(time.Now().Unix() - event.Timestamp)
			if timestampSec < 0 {
				timestampSec = 0
			}

			result[i] = map[string]interface{}{
				"type":      event.Type,
				"seqNum":    seqNum,
				"timestamp": timestampSec,
				"data":      e.sanitizeForAMF3(event.Data),
			}
		}
		return result
	case types.Event:
		// handle single WebAPIEvent
		// AMF3 has a 29-bit limit for integers
		seqNum := int(v.SeqNum % (1 << 29))
		// convert timestamp to seconds ago to keep it small
		timestampSec := int(time.Now().Unix() - v.Timestamp)
		if timestampSec < 0 {
			timestampSec = 0
		}

		return map[string]interface{}{
			"type":      v.Type,
			"seqNum":    seqNum,
			"timestamp": timestampSec,
			"data":      e.sanitizeForAMF3(v.Data),
		}
	default:
		// for other types,
		// use reflection to check if it's a struct and convert to map
		rv := reflect.ValueOf(data)
		if rv.Kind() == reflect.Struct {
			return e.structToMap(rv)
		}
		return data
	}
}

// DetectAMFVersion determines which AMF version to use based on the request.
func DetectAMFVersion(r *http.Request) AMFVersion {
	if r == nil {
		return AMF3
	}

	// check query parameter first (highest priority)
	format := strings.ToLower(r.URL.Query().Get("f"))
	switch format {
	case "amf3":
		return AMF3
	case "amf":
		// default to AMF3 for modern clients (Gromit expects AMF3)
		return AMF3
	}

	// check Accept header for version hint
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "amf3") || strings.Contains(accept, "AMF3") {
		return AMF3
	}

	if strings.Contains(accept, "amf") || strings.Contains(accept, "AMF") {
		return AMF3 // Default to AMF3 for AMF requests
	}

	// check Content-Type header (for POST requests)
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "amf3") || strings.Contains(contentType, "AMF3") {
		return AMF3
	}

	if strings.Contains(contentType, "amf") || strings.Contains(contentType, "AMF") {
		return AMF3 // Default to AMF3 for AMF requests
	}

	// default to AMF3 for modern clients
	return AMF3
}
