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

// RoastOSCARPassword roasts an OSCAR client password.
func RoastOSCARPassword(roastedPass []byte) []byte {
	var roastTable = []byte{
		0xF3, 0x26, 0x81, 0xC4, 0x39, 0x86, 0xDB, 0x92,
		0x71, 0xA3, 0xB9, 0xE6, 0x53, 0x7A, 0x95, 0x7C,
	}
	return roastPass(roastedPass, roastTable)
}

// RoastKerberosPassword roasts a Kerberos client password.
func RoastKerberosPassword(roastedPass []byte) []byte {
	var roastTable = []byte{
		0x76, 0x91, 0xc5, 0xe7, 0xd0, 0xd9, 0x95, 0xdd,
		0x9e, 0x2F, 0xea, 0xd8, 0x6B, 0x21, 0xc2, 0xbc,
	}
	return roastPass(roastedPass, roastTable)
}

// roastPass toggles obfuscation/deobfuscates of roastedPass.
func roastPass(roastedPass []byte, roastTable []byte) []byte {
	clearPass := make([]byte, len(roastedPass))
	for i := range roastedPass {
		clearPass[i] = roastedPass[i] ^ roastTable[i%len(roastTable)]
	}
	return clearPass
}
