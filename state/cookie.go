package state

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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

func (h *hmacToken) hash(key []byte) {
	hs := hmac.New(sha256.New, key)
	if _, err := hs.Write(h.Data); err != nil {
		// according to Hash interface, Write() should never return an error
		panic("unable to compute hmac token")
	}
	h.Sig = hs.Sum(nil)
}

func (h *hmacToken) validate(key []byte) bool {
	cp := *h
	cp.hash(key)
	return hmac.Equal(h.Sig, cp.Sig)
}
