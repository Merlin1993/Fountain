package crypto

import (
	"crypto/ecdsa"
	"testing"
	"witCon/common"
	"witCon/log"
)

func TestSig(t *testing.T) {
	sk, _ := GenerateKey()
	sig, _ := Sign(common.Hash{}.Bytes(), sk)
	pb, _ := SigToPub(common.Hash{}.Bytes(), sig)
	t.Log(PubKeyToAddress(*pb).String(), PubKeyToAddress(sk.PublicKey).String())
}

func TestSchnorrSig(t *testing.T) {
	msg := make([]common.Hash, 10)
	for i := 0; i < 10; i++ {
		msg[i] = HashSum([]byte{byte(i)})
	}

	sk, _ := GenerateKey()
	result, _ := SignSchnorr(sk.D, msg[0])
	succ, err := VerifySchnorrForPK(&sk.PublicKey, msg[0], result)
	if err != nil {
		log.Error("verify fail", "err", err)
		return
	}
	if succ {
		t.Log("succ")
	} else {
		t.Log("fail")
	}
}

func TestBatchSchnorrSig(t *testing.T) {
	msg := make([]common.Hash, 10)
	sk := make([]*ecdsa.PrivateKey, 10)
	result := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		msg[i] = HashSum([]byte{byte(i)})
		sk[i], _ = GenerateKey()
		result[i], _ = SignSchnorr(sk[i].D, msg[i])
	}
	//result[8][0] = 0xff

	batchLen := 10
	pks := make([][33]byte, batchLen)
	msgs := make([][32]byte, batchLen)
	sigs := make([][64]byte, batchLen)
	for i := 0; i < batchLen; i++ {
		pks[i] = SchnorrPk(CompressPubKey(&sk[i].PublicKey))
		msgs[i] = msg[i]
		sigs[i] = SchnorrSig(result[i])
	}

	succ, err := VerifyBatchSchnorr(pks, msgs, sigs)
	if err != nil {
		t.Log("verify fail:", err.Error())
		return
	}
	if succ {
		t.Log("succ")
	} else {
		t.Log("fail")
	}
}
