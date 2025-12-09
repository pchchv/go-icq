package wire

import (
	"crypto/md5"
	"io"
)

// WeakMD5PasswordHash hashes password and authKey for AIM v3.5-v4.7.
//
//goland:noinspection ALL
func WeakMD5PasswordHash(pass, authKey string) []byte {
	hash := md5.New()
	io.WriteString(hash, authKey)
	io.WriteString(hash, pass)
	io.WriteString(hash, "AOL Instant Messenger (SM)")
	return hash.Sum(nil)
}

// StrongMD5PasswordHash hashes password and authKey for AIM v4.8+.
//
//goland:noinspection ALL
func StrongMD5PasswordHash(pass, authKey string) []byte {
	top := md5.New()
	io.WriteString(top, pass)
	bottom := md5.New()
	io.WriteString(bottom, authKey)
	bottom.Write(top.Sum(nil))
	io.WriteString(bottom, "AOL Instant Messenger (SM)")
	return bottom.Sum(nil)
}

// roastPass toggles obfuscation/deobfuscates of roastedPass.
func roastPass(roastedPass []byte, roastTable []byte) []byte {
	clearPass := make([]byte, len(roastedPass))
	for i := range roastedPass {
		clearPass[i] = roastedPass[i] ^ roastTable[i%len(roastTable)]
	}
	return clearPass
}
