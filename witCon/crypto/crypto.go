package crypto

import (
	"crypto/sha256"
	"witCon/common"
)

var (
	INT_SIG_LEN = 65
)

type PrivateKey struct {
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

func Sha256(data []byte) common.Hash {
	hash := common.Hash{}
	sha := sha256.New()
	sha.Write(data)
	sha.Sum(hash[:0])
	return hash
}

func RecoverCA(sigHash common.Hash, signature []byte) (common.Address, error) {
	pub, err := EcRecover(sigHash[:], signature)
	if err != nil {
		return common.EmptyAddress, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.EmptyAddress, ErrInvalidPubKey
	}
	var addr common.Address
	hash := common.Hash{}
	sha := sha256.New()
	sha.Write(pub[1:])
	sha.Sum(hash[:0])
	copy(addr[:], hash[12:])
	return addr, nil
}
