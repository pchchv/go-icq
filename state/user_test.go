package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayScreenName_ValidateAIMHandle(t *testing.T) {
	tests := []struct {
		name    string
		input   DisplayScreenName
		wantErr error
	}{
		{"Valid handle no spaces", "User123", nil},
		{"Valid handle with min character count and space", "U SR", nil},
		{"Valid handle with min character including letters and numbers", "dj3520", nil},
		{"Valid handle with max character count", "JustTheRightSize", nil},
		{"Valid handle with max character count and spaces", "Just   RightSize", nil},
		{"Too short", "Us", ErrAIMHandleLength},
		{"Too short due to spaces", "U S", ErrAIMHandleLength},
		{"Too long", "ThisIsAReallyLongScreenName", ErrAIMHandleLength},
		{"Too many spaces", "User           123 ", ErrAIMHandleLength},
		{"Starts with number", "1User", ErrAIMHandleInvalidFormat},
		{"Ends with space", "User123 ", ErrAIMHandleInvalidFormat},
		{"Contains invalid character", "User@123", ErrAIMHandleInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.ValidateAIMHandle()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr, "ValidateAIMHandle() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				assert.NoError(t, err, "ValidateAIMHandle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDisplayScreenName_ValidateICQHandle(t *testing.T) {
	tests := []struct {
		name    string
		input   DisplayScreenName
		wantErr error
	}{
		{"Valid UIN", "123456", nil},
		{"Too low", "9999", ErrICQUINInvalidFormat},
		{"Too high", "2147483647", ErrICQUINInvalidFormat},
		{"Non-numeric", "abcd", ErrICQUINInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.ValidateUIN()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr, "ValidateUIN() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				assert.NoError(t, err, "ValidateUIN() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
