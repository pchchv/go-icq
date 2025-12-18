package state

import (
	"crypto/rand"
	"fmt"
	"io"
)

type HMACCookieBaker struct {
	key []byte
}

func NewHMACCookieBaker() (HMACCookieBaker, error) {
	cb := HMACCookieBaker{}
	cb.key = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, cb.key); err != nil {
		return cb, fmt.Errorf("cannot generate random HMAC key: %w", err)
	}
	return cb, nil
}

type hmacToken struct {
	Data []byte `oscar:"len_prefix=uint16"`
	Sig  []byte `oscar:"len_prefix=uint16"`
}
