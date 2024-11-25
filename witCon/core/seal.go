package core

import (
	"crypto/ecdsa"
	"witCon/common"
	"witCon/crypto"
	"witCon/log"
)

type Seal struct {
	sk   *ecdsa.PrivateKey
	addr common.Address
}

func NewSeal(sk *ecdsa.PrivateKey) *Seal {
	s := &Seal{sk: sk}
	s.addr = crypto.PubKeyToAddress(sk.PublicKey)
	return s
}

func (s *Seal) Signature(hash common.Hash) ([]byte, error) {
	sig, err := crypto.Sign(hash.Bytes(), s.sk)
	if err != nil {
		log.Error("sign err", "err", err)
		return nil, err
	}
	return sig, nil
}

func (s *Seal) Coinbase() common.Address {
	return s.addr
}

func (s *Seal) Verify(sig []byte, hash common.Hash, addr common.Address) bool {
	pub, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		log.Error("Verify err", "err", err)
		return false
	}
	a := crypto.PubKeyToAddress(*pub)
	if a != addr {
		log.Error("Verify err", "a", a.String(), "addr", addr.String())
	}
	return a == addr
}
