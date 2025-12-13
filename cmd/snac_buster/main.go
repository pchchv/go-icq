package main

import (
	"bytes"
	"fmt"

	"github.com/pchchv/go-icq/wire"
)

func printByteSlice(data []byte) {
	fmt.Print("[]byte{")
	for i, b := range data {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("0x%02X", b)
	}
	fmt.Println("}")
}

func main() {
	b := []byte{}
	flap := wire.FLAPFrame{}
	wire.UnmarshalBE(&flap, bytes.NewReader(b))

	rd := bytes.NewBuffer(flap.Payload)
	snac := wire.SNACFrame{}
	wire.UnmarshalBE(&snac, rd)

	printByteSlice(rd.Bytes())

	snacBody := wire.SNAC_0x01_0x0F_OServiceUserInfoUpdate{}
	wire.UnmarshalBE(&snacBody, rd)
	fmt.Println(snacBody)
}
