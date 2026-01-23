package handlers

import (
	"testing"
	"time"

	goAMF3 "github.com/pchchv/amf"
)

func TestAMFEncoderBasicTypes(t *testing.T) {
	encoder := NewAMFEncoder(nil)
	tests := []struct {
		name    string
		input   interface{}
		version AMFVersion
		wantErr bool
	}{
		{"String AMF3", "hello world", AMF3, false},
		{"Number AMF3", 42, AMF3, false},
		{"Float AMF3", 3.14159, AMF3, false},
		{"Boolean AMF3", false, AMF3, false},
		{"Null AMF3", nil, AMF3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := encoder.EncodeAMF(tt.input, tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EncodeAMF() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(data) == 0 {
				t.Fatal("EncodeAMF() returned empty data")
			}

			// try to decode the data to verify it's valid AMF3
			if !tt.wantErr {
				decoded := goAMF3.DecodeAMF3(data)
				if decoded == nil {
					t.Fatalf("Failed to decode AMF3 data: got nil result")
				}
			}
		})
	}
}

func TestAMFEncoderComplexTypes(t *testing.T) {
	encoder := NewAMFEncoder(nil)
	tests := []struct {
		name    string
		input   interface{}
		version AMFVersion
	}{
		{
			name: "Map",
			input: map[string]interface{}{
				"name":   "John Doe",
				"age":    30,
				"active": true,
			},
			version: AMF3,
		},
		{
			name: "Array",
			input: []interface{}{
				"item1",
				42,
				true,
				nil,
			},
			version: AMF3,
		},
		{
			name: "BaseResponse",
			input: BaseResponse{
				Response: ResponseBody{
					StatusCode: 200,
					StatusText: "OK",
					Data: map[string]interface{}{
						"user":   "testuser",
						"online": true,
						"buddies": []interface{}{
							"friend1",
							"friend2",
						},
					},
				},
			},
			version: AMF3,
		},
		{
			name: "ErrorResponse",
			input: ErrorResponse{
				Response: struct {
					StatusCode int    `json:"statusCode" xml:"statusCode"`
					StatusText string `json:"statusText" xml:"statusText"`
				}{
					StatusCode: 404,
					StatusText: "Not Found",
				},
			},
			version: AMF3,
		},
		{
			name: "Time",
			input: map[string]interface{}{
				"timestamp": time.Now(),
				"name":      "Event",
			},
			version: AMF3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := encoder.EncodeAMF(tt.input, tt.version)
			if err != nil {
				t.Fatalf("EncodeAMF() error = %v", err)
			}

			if len(data) == 0 {
				t.Fatal("EncodeAMF() returned empty data")
			}

			// verify the data is valid AMF
			decoded := goAMF3.DecodeAMF3(data)

			if decoded == nil {
				t.Fatalf("Failed to decode AMF data: got nil result")
			}

			// log the size for performance comparison
			t.Logf("%s: %d bytes", tt.name, len(data))
		})
	}
}
