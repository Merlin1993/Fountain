package common

import (
	"fmt"
	"math/big"
	"reflect"
	"witCon/common/hexutil"
)

const HashLength = 32

var EmptyHash = Hash{}

var hashT = reflect.TypeOf(Hash{})

type Hash [HashLength]byte

func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}
	copy(h[HashLength-len(b):], b)
}

func (h Hash) Bytes() []byte { return h[:] }

func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

func (h Hash) String() string {
	return h.Hex()
}

func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

type Hashes []Hash

func NewHashes(in []byte) Hashes {
	len := len(in) / HashLength
	hs := make([]Hash, len)
	for i := 0; i < len; i++ {
		hs[i] = Hash{}
		hs[i].SetBytes(in[i*HashLength : (i+1)*HashLength])
	}
	return hs
}

func (hs Hashes) GetByte() []byte {
	out := make([]byte, len(hs)*HashLength)
	for i, hash := range hs {
		copy(out[i*HashLength:(i+1)*HashLength], hash[:])
	}
	return out
}

func HashSlice2Interface(hashes []Hash) []interface{} {
	contractsTransit := make([]interface{}, len(hashes))
	for index, value := range hashes {
		contractsTransit[index] = value
	}
	return contractsTransit
}
