package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"io/ioutil"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/crypto"
)

type sigTxs struct {
	From      common.Address
	To        common.Address
	Number    uint64
	Amount    *big.Int
	PublicKey [33]byte
	Signature []byte
}

func TestSignatureTxs(t *testing.T) {
	index := 0
	stxs := readFile(index)
	//stxs = stxs[:100]
	txs := make([]*sigTxs, len(stxs))
	for i, tx := range stxs {
		sk, _ := crypto.GenerateKey()
		data := append([]byte{0, 0, 0, 0, 0, 0, 0, 2}, tx.To.Bytes()...)
		data = append(data, tx.Amount.Bytes()...)
		data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, data})
		if err != nil {
			t.Log("encode tx fail:", err)
			return
		}
		hash := crypto.HashSum(data)
		result, _ := crypto.SignSchnorr(sk.D, hash)
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = &sigTxs{
			tx.From, tx.To, tx.Number, tx.Amount, crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey)), result,
		}
	}
	txpath := fmt.Sprintf("%s\\%v", "E:\\ethdata\\sig\\", index)
	if _, err := os.Stat(txpath); os.IsNotExist(err) {
		b, err := rlp.EncodeToBytes(txs)

		err = ioutil.WriteFile(txpath, b, 0777)
		if err != nil {
			fmt.Println("文件写入失败", err)
			return
		}
		fmt.Println("文件写入成功", err)
		return
	}
}

func TestECDSASignatureTxs(t *testing.T) {
	index := 16000000
	stxs := readFile(index)
	//stxs = stxs[:100]
	txs := make([]*sigTxs, len(stxs))
	for i, tx := range stxs {
		sk, _ := crypto.GenerateKey()
		data := append([]byte{0, 0, 0, 0, 0, 0, 0, 2}, tx.To.Bytes()...)
		data = append(data, tx.Amount.Bytes()...)
		data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, data})
		if err != nil {
			t.Log("encode tx fail:", err)
			return
		}
		hash := crypto.HashSum(data)
		result, _ := crypto.Sign(hash[:], sk)
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = &sigTxs{
			tx.From, tx.To, tx.Number, tx.Amount, crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey)), result,
		}
	}
	txpath := fmt.Sprintf("%s\\slice%v", common.EthTxPath, index)
	//必须删掉文件才能重新生成
	if _, err := os.Stat(txpath); os.IsNotExist(err) {
		f, err := os.OpenFile(txpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
		if err != nil {
			fmt.Println("Failed to open file", err)
			return
		}
		defer f.Close()
		for i := 0; i*100000 < len(txs)-100000; i++ {
			b, err := rlp.EncodeToBytes(txs[i*100000 : (i+1)*100000])
			fmt.Printf("size:%v", len(b))
			_, err = f.Write(b)
			if err != nil {
				fmt.Println("文件写入失败", err)
				return
			}
		}

		fmt.Println("文件写入成功", err)
		return
	}
}

func TestECDSASignature(t *testing.T) {
	index := 16000000
	stxs := readFile(index)
	tx0 := stxs[0]
	sk, _ := crypto.GenerateKey()
	data1 := append([]byte{0, 0, 0, 0, 0, 0, 0, 2}, tx0.To.Bytes()...)
	data1 = append(data1, tx0.Amount.Bytes()...)
	data0, err := rlp.EncodeToBytes([]interface{}{tx0.From, tx0.Number, data1})
	if err != nil {
		t.Log("encode tx fail:", err)
		return
	}
	hash1 := crypto.HashSum(data0)
	result, _ := crypto.Sign(hash1[:], sk)
	// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
	cplk := crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey))
	sigTx := &sigTxs{
		tx0.From, tx0.To, tx0.Number, tx0.Amount, cplk, result,
	}
	tpk, err := crypto.SigToPub(hash1[:], result)
	tpks := crypto.CompressPubKey(tpk)
	t.Log("tpk", "pk", hex.EncodeToString(tpks), hash1.String(), hex.EncodeToString(result))

	ss := block.NewTx(sigTx.From, sigTx.To, sigTx.Amount, sigTx.Number)
	ss.SetSig(sigTx.PublicKey, sigTx.Signature)

	d2, err := rlp.EncodeToBytes([]interface{}{ss.From, ss.Number, ss.Data})
	if err != nil {
		t.Log("encode err", err.Error())
		return
	}
	hash2 := crypto.HashSum(d2)
	pk, err := crypto.SigToPub(hash2[:], ss.Sig[33:])
	pks := crypto.CompressPubKey(pk)
	if err != nil || !bytes.Equal(pks, ss.Sig[:33]) {
		t.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks), "tx", hex.EncodeToString(ss.Sig[:33]))
		t.Error("pk fail", hash2.String(), hex.EncodeToString(ss.Sig[33:]))
		return
	}
}

func TestEcdsaSpeed(t *testing.T) {
	count := atomic.Int32{}
	count.Store(0)

	sk, _ := crypto.GenerateKey()
	hash := crypto.HashSum([]byte{0x01, 0x02})
	result, _ := crypto.Sign(hash[:], sk)
	pks := crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey))

	verifySigPool, _ := ants.NewPool(16)

	tj := 100
	ti := 500
	go func() {
		for {
			var wg sync.WaitGroup
			for j := 0; j < tj; j++ {
				wg.Add(1)
				verifySigPool.Submit(func() {
					for i := 0; i < ti; i++ {
						tpk, err := crypto.SigToPub(hash[:], result)
						tpks := crypto.CompressPubKey(tpk)
						if err != nil || !bytes.Equal(tpks, pks[:]) {
							t.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks[:]), "tx", hex.EncodeToString(tpks))
							return
						}
						count.Add(1)
					}
					wg.Done()
				})
			}
			wg.Wait()
		}
	}()

	var last int32
	timer := time.NewTimer(0 * time.Second)
	//<-timer.C
	for {
		select {
		case <-timer.C:
			tl := last
			last = count.Load()
			t.Log("timerLoop", "count", last-tl)
			timer.Reset(1 * time.Second)
		}
	}

}

func TestShnnorSpeed(t *testing.T) {
	count := atomic.Int32{}
	count.Store(0)

	sk, _ := crypto.GenerateKey()
	hash := crypto.HashSum([]byte{0x01, 0x02})
	result, _ := crypto.SignSchnorr(sk.D, hash)
	pks := crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey))

	verifySigPool, _ := ants.NewPool(16)

	tj := 100
	ti := 500
	go func() {
		for {
			var wg sync.WaitGroup
			for j := 0; j < tj; j++ {
				wg.Add(1)
				verifySigPool.Submit(func() {
					for i := 0; i < ti; i++ {
						tpk, err := crypto.VerifySchnorr(pks[:], hash, result)
						if !tpk || err != nil {
							t.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks[:]))
							return
						}
						count.Add(1)
					}
					wg.Done()
				})
			}
			wg.Wait()
		}
	}()

	var last int32
	timer := time.NewTimer(0 * time.Second)
	//<-timer.C
	for {
		select {
		case <-timer.C:
			tl := last
			last = count.Load()
			t.Log("timerLoop", "count", last-tl)
			timer.Reset(1 * time.Second)
		}
	}
}

//func TestOtherEcdsaSpeed(t *testing.T) {
//	count := atomic.Int32{}
//	count.Store(0)
//
//	sk, _ := crypto.GenerateKey()
//	hash := crypto.HashSum([]byte{0x01, 0x02})
//	result, _ := ecdsa.Sign().SignSchnorr(sk.D, hash)
//	pks := crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey))
//
//	verifySigPool, _ := ants.NewPool(16)
//
//	tj := 100
//	ti := 500
//	go func() {
//		for {
//			var wg sync.WaitGroup
//			for j := 0; j < tj; j++ {
//				wg.Add(1)
//				verifySigPool.Submit(func() {
//					for i := 0; i < ti; i++ {
//						tpk, err := crypto.VerifySchnorr(pks[:], hash, result)
//						if !tpk || err != nil {
//							t.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks[:]))
//							return
//						}
//						count.Add(1)
//					}
//					wg.Done()
//				})
//			}
//			wg.Wait()
//		}
//	}()
//
//	var last int32
//	timer := time.NewTimer(0 * time.Second)
//	//<-timer.C
//	for {
//		select {
//		case <-timer.C:
//			tl := last
//			last = count.Load()
//			t.Log("timerLoop", "count", last-tl)
//			timer.Reset(1 * time.Second)
//		}
//	}
//}
