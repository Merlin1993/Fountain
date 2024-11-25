package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"sync/atomic"
	"witCon/common"
	"witCon/common/hexutil"
	"witCon/common/math"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
	emptyHashCache atomic.Value
)

// 国际加密算法
type InternationalCrypto struct {
	emptyHashCache atomic.Value
}

func HashSumByte(data ...[]byte) []byte {
	hash256 := sha256.New()
	for _, b := range data {
		hash256.Write(b)
	}
	return hash256.Sum(nil)
}

func EncodeHash(fun func(io.Writer)) (h common.Hash) {
	hash := sha256.New()
	fun(hash)
	hash.Sum(h[:0])
	return h
}

func HashSum(data ...[]byte) (h common.Hash) {
	hash256 := sha256.New()
	for _, b := range data {
		hash256.Write(b)
	}
	hash256.Sum(h[:0])
	return h
}

var CurveI = btcec.S256()

// VerifyPKHash 使用 btcec 的优化方法验证签名
func VerifyPKHash(pubKey *btcec.PublicKey, hash, sig []byte) bool {
	// 尝试解析公钥
	//pubKey, err := btcec.ParsePubKey(pk, CurveI)
	//if err != nil {
	//	// 公钥解析失败，直接返回 false
	//	return false
	//}

	// 校验签名长度是否为 2*Nlen
	//Nlen := (CurveI.Params().BitSize + 7) >> 3
	//if len(sig) != 2*Nlen {
	//	return false
	//}

	// 将签名拆分为 r 和 s
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])

	// 使用 btcec 的 VerifySignature 方法直接验证签名
	return ecdsa.Verify(pubKey.ToECDSA(), hash, r, s)
}

func EmptyHash() (h common.Hash) {
	if hash := emptyHashCache.Load(); (hash != nil && hash != common.Hash{}) {
		return hash.(common.Hash)
	}
	v := HashSum(nil)
	emptyHashCache.Store(v)
	return v
}

func SavePrivateKey(file string, key *ecdsa.PrivateKey) error {
	k := hex.EncodeToString(FromECDSA(key))
	return ioutil.WriteFile(file, []byte(k), 0600)
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(Curve(), rand.Reader)
}

func LoadPrivateKey(file string) (*ecdsa.PrivateKey, error) {
	buf := make([]byte, 64)
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	if _, err := io.ReadFull(fd, buf); err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}
	return toECDSA(key, true)
}

// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexKey string) (*ecdsa.PrivateKey, error) {
	b, err := hexutil.Decode(hexKey)
	if err != nil {
		return nil, errInvalidHexPrivateKey.ErrorOf(err.Error())
	}
	return toECDSA(b, true)
}

func ToECDSAUnsafe(d []byte) *ecdsa.PrivateKey {
	priv, _ := toECDSA(d, false)
	return priv
}

func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = Curve()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, errInvalidPrivateKeyLength.ErrorOf(priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

func FromECDSA(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

func UnmarshalPubKey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(Curve(), pub)
	if x == nil {
		return nil, errInvalidPubKey
	}
	return &ecdsa.PublicKey{Curve: Curve(), X: x, Y: y}, nil
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(Curve(), pub.X, pub.Y)
}

func PubKeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := FromECDSAPub(&p)
	return common.BytesToAddress(HashSumByte(pubBytes[1:])[12:])
}

func ValidateSignatureValues(v byte, r, s *big.Int) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	//if s.Cmp(secp256k1halfN) > 0 {
	//	log.Error("s > secp256k1halfN ")
	//	return false
	//}
	return r.Cmp(secp256k1N) < 0 && s.Cmp(secp256k1N) < 0 && (v == 0 || v == 1)
}

// 32 + 32 + 1
func ExtraSeal() int {
	return 65
}
