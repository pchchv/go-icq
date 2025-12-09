package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeakMD5PasswordHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		authKey  string
		want     []byte
	}{
		{
			name:     "empty password and auth key",
			password: "",
			authKey:  "",
			want:     []byte{0x13, 0xfd, 0x0b, 0x9e, 0x89, 0xf4, 0xb8, 0x36, 0xa7, 0x65, 0x8b, 0x9d, 0xca, 0xad, 0x2a, 0xd4},
		},
		{
			name:     "password and auth key",
			password: "password123",
			authKey:  "authkey456",
			want:     []byte{0x04, 0x79, 0x63, 0x82, 0x0d, 0xa7, 0xbb, 0xfe, 0x6a, 0x9b, 0x41, 0xa4, 0x5c, 0x47, 0xcb, 0xcb},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WeakMD5PasswordHash(tt.password, tt.authKey)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStrongMD5PasswordHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		authKey  string
		want     []byte
	}{
		{
			name:     "empty password and auth key",
			password: "",
			authKey:  "",
			want:     []byte{0x1f, 0xa2, 0xb6, 0x99, 0x59, 0x84, 0xb0, 0x14, 0x68, 0xa3, 0x7c, 0x77, 0x42, 0x90, 0x0a, 0xc9},
		},
		{
			name:     "password and auth key",
			password: "password123",
			authKey:  "authkey456",
			want:     []byte{0xb9, 0x07, 0x91, 0xcc, 0xcb, 0x5c, 0x57, 0x71, 0xbd, 0xcb, 0xc9, 0x39, 0x82, 0xf7, 0x94, 0x84},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StrongMD5PasswordHash(tt.password, tt.authKey)
			assert.Equal(t, tt.want, got)
		})
	}
}
