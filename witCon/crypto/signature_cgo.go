package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"
	"witCon/common"
	"witCon/common/math"
	"witCon/crypto/schnorr"
	secp256k1 "witCon/crypto/secp256k1"
	"witCon/log"
)

func EcRecover(hash, sig []byte) ([]byte, error) {
	sig = sig[:INT_SIG_LEN]
	return secp256k1.RecoverPubkey(hash, sig)
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	s, err := EcRecover(hash, sig)
	if err != nil {
		return nil, err
	}

	x, y := elliptic.Unmarshal(Curve(), s)
	return &ecdsa.PublicKey{Curve: Curve(), X: x, Y: y}, nil
}

func Sign(hash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	secKey := math.PaddedBigBytes(prv.D, prv.Params().BitSize/8)
	defer zeroBytes(secKey)
	return secp256k1.Sign(hash, secKey)
}

func VerifySignature(pubKey, hash, signature []byte) bool {
	return secp256k1.VerifySignature(pubKey, hash, signature)
}

func Curve() elliptic.Curve {
	return secp256k1.S256()
}

func DecompressPubKey(pubKey []byte) (*ecdsa.PublicKey, error) {
	x, y := secp256k1.DecompressPubkey(pubKey)
	if x == nil {
		return nil, fmt.Errorf("invalid public key")
	}
	return &ecdsa.PublicKey{X: x, Y: y, Curve: Curve()}, nil
}

func CompressPubKey(pubKey *ecdsa.PublicKey) []byte {
	return secp256k1.CompressPubkey(pubKey.X, pubKey.Y)
}

func VerifySchnorrForPK(key *ecdsa.PublicKey, hash common.Hash, sig []byte) (bool, error) {
	pk := CompressPubKey(key)
	return VerifySchnorr(pk, hash, sig)
}

func VerifySchnorr(pk []byte, hash common.Hash, sig []byte) (bool, error) {
	if len(pk) < 33 {
		return false, fmt.Errorf("pk len is fail")
	}
	if len(sig) < 64 {
		return false, fmt.Errorf("sig len is fail")
	}
	var pk33 [33]byte
	var sig64 [64]byte
	copy(pk33[:], pk)
	copy(sig64[:], sig)
	return schnorr.Verify(pk33, hash, sig64)
}

func SchnorrPk(pk []byte) [33]byte {
	var pk33 [33]byte
	copy(pk33[:], pk)
	return pk33
}

func SchnorrSig(sig []byte) [64]byte {
	var sig64 [64]byte
	copy(sig64[:], sig)
	return sig64
}

func SchnorrMsg(hash common.Hash) [32]byte {
	return hash
}

func VerifyBatchSchnorr(key [][33]byte, msg [][32]byte, sig [][64]byte) (bool, error) {
	return schnorr.BatchVerify(key, msg, sig)
}

func SignSchnorr(privateKey *big.Int, hash common.Hash) ([]byte, error) {
	sig, err := schnorr.Sign(privateKey, hash)
	if err != nil {
		log.Error("sign schnorr fail.", "err", err)
		return nil, err
	}
	return sig[:], err
}

func GenerateSchnorrKey() (privateKey *big.Int, publicKey *big.Int) {
	return
}
