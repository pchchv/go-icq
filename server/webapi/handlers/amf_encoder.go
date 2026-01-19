package handlers

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"
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

// convertToMap converts any data to a map structure for AMF3
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

// responseBodyToMap converts ResponseBody to AMF3-compatible map
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
