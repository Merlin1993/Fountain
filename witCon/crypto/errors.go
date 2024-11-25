package crypto

import (
	"witCon/common/zerror"
)

var (
	ErrInvalidSig              = zerror.New("签名有误，请检查v,r,s字段（sign字段）", "invalid transaction v, r, s values", 4000)
	ErrInvalidPubKey           = zerror.New("公钥格式有误", "invalid public key crypto", 4001)
	errInvalidPubKey           = zerror.New("公钥格式有误", "invalid secp256k1 public key", 4002)
	errInvalidPrivateKeyLength = zerror.NewErrorParams("私钥长度有误，私钥长度应为 %d bit位", "invalid length, need %d bits", 4003)
	errInvalidHexPrivateKey    = zerror.NewErrorParams("私钥的编码无法用hex编码解析：%s", "invalid hex string：%s", 4004)
)
