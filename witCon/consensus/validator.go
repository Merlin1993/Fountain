package consensus

import (
	"bytes"
	"encoding/hex"
	"github.com/panjf2000/ants/v2"
	"sync"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/log"
)

func VerifyBatchTxSig(txs []*block.Transaction) (bool, error) {
	batchLen := len(txs)
	if batchLen == 0 {
		return true, nil
	}
	pks := make([][33]byte, batchLen)
	msgs := make([][32]byte, batchLen)
	sigs := make([][64]byte, batchLen)
	for i, tx := range txs {
		pks[i] = crypto.SchnorrPk(tx.Sig[:33])
		data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
		if err != nil {
			return false, err
		}
		hash := crypto.HashSum(data)
		msgs[i] = hash
		sigs[i] = crypto.SchnorrSig(tx.Sig[33:])
	}
	return crypto.VerifyBatchSchnorr(pks, msgs, sigs)
}

func VerifyTxsSig(txs []*block.Transaction) (bool, error) {
	for _, tx := range txs {
		data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
		if err != nil {
			return false, err
		}
		hash := crypto.HashSum(data)
		msgs := hash
		succ, err := crypto.VerifySchnorr(tx.Sig[:33], msgs, tx.Sig[33:])
		if !succ {
			return succ, err
		}
	}
	return true, nil
}

func VerifyTxsEcdsaSig(txs []*block.Transaction) (bool, error) {
	for _, tx := range txs {
		data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
		if err != nil {
			log.Error("encode fail", "err", err.Error())
			return false, err
		}
		hash := crypto.HashSum(data)
		pk, err := crypto.SigToPub(hash[:], tx.Sig[33:])
		pks := crypto.CompressPubKey(pk)
		if err != nil || !bytes.Equal(pks, tx.Sig[:33]) {
			log.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks), "tx", hex.EncodeToString(tx.Sig[:33]))
			return false, err
		}
	}
	return true, nil
}

func VerifyTxsEcdsaSigMul(txs []*block.Transaction, verifySigPool *ants.Pool) (bool, error) {
	//shardSize := 100
	//shardlen := len(txs)/shardSize + 1

	shardlen := 5 * int(common.SignVerifyCore)
	shardSize := len(txs)/shardlen + 1
	var wg sync.WaitGroup
	var verr error
	t1 := time.Now().UnixMilli()
	for i := 0; i < shardlen; i++ {
		wg.Add(1)
		start := i * shardSize
		end := (i + 1) * shardSize
		if end > len(txs) {
			end = len(txs)
		}
		verifySigPool.Submit(func() {
			defer wg.Done()
			for j := start; j < end; j++ {
				tx := txs[j]
				data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
				if err != nil {
					log.Error("encode fail", "err", err.Error())
					verr = err
					return
				}
				hash := crypto.HashSum(data)
				pk, err := crypto.SigToPub(hash[:], tx.Sig[33:])
				pks := crypto.CompressPubKey(pk)
				if err != nil || !bytes.Equal(pks, tx.Sig[:33]) {
					log.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks), "tx", hex.EncodeToString(tx.Sig[:33]))
					return
				}
			}
		})
	}
	wg.Wait()

	t2 := time.Now().UnixMilli()
	if len(txs) > 0 {
		log.Warn("verify mul", "len", len(txs), "ms", t2-t1)
	}
	if verr != nil {
		return false, verr
	}
	return true, nil
}
