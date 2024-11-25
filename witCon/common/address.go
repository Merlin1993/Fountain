package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

const (
	AddrVersion = 1
	AddrLength  = 20
	AddrTitle   = "addr"
)

var (
	addressT     = reflect.TypeOf(Address{})
	EmptyAddress = Address{}
)

type Address [AddrLength]byte

func (a Address) Bytes() []byte { return a[:] }

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddrLength:]
	}
	copy(a[AddrLength-len(b):], b)
}

func (a Address) Hex() string {
	unCheckSummed := hex.EncodeToString(a[:])
	sha := sha3.NewLegacyKeccak256()
	sha.Write([]byte(unCheckSummed))
	hash := sha.Sum(nil)

	result := []byte(unCheckSummed)
	for i := 0; i < len(result); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return "0x" + string(result)
}

func (a Address) Base58CheckSum() string {
	return base58.CheckEncode(a[:], AddrVersion)
}

func (a Address) String() string {
	return strconv.Itoa(int(Shard(a))) + "_" + a.Base58CheckSum()
}

func (a Address) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), a[:])
}

func (a Address) IsEmpty() bool {
	return a == EmptyAddress
}

func (a *Address) UnmarshalText(input []byte) error {
	address, err := Base58ToAddress(string(input[:]))
	if err != nil {
		return err
	}
	a.SetBytes(address.Bytes())
	return nil
}

func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *Address) UnmarshalJSON(input []byte) error {
	addrStr := ""
	err := json.Unmarshal(input, &addrStr)
	if err != nil {
		return err
	}
	address, err := Base58ToAddress(addrStr)
	if err != nil {
		return err
	}
	a.SetBytes(address.Bytes())
	return nil
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BigToAddress returns Address with byte values of b.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

func Base58ToAddress(s string) (Address, error) {
	elem := strings.SplitN(s, "_", 2)
	if len(elem) != 2 {
		return Address{}, ErrAddrFormat
	}
	//if elem[0] != AddrTitle {
	//	return Address{}, ErrAddrFormat
	//}
	dec, version, err := base58.CheckDecode(elem[1])
	if version != AddrVersion || err != nil {
		return Address{}, ErrAddrFormat
	}
	return BytesToAddress(dec), nil
}

func IsContain(items []Address, item Address) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
