package core

import (
	"github.com/panjf2000/ants/v2"
	"sync"
	"witCon/common"
	"witCon/common/block"
	"witCon/consensus"
	"witCon/log"
)

// 状态是做加法，交易池是加法和减法
type TxPool struct {
	pending     []*block.Transaction
	currentAddr common.Address
	parentNonce map[common.Hash]uint64
	lock        sync.RWMutex

	sigPool *ants.Pool
	//dirty    map[common.Hash]txState //未确定的区块的状态
	//dirtyNum map[uint64][]common.Hash
}

type txState struct {
	parent    common.Hash
	addrNonce map[common.Address]uint64
}

func NewTxPool() *TxPool {

	p := TxPool{
		pending:     make([]*block.Transaction, 0, 100000000),
		parentNonce: make(map[common.Hash]uint64),
		//lock:        nil,
	}
	p.sigPool, _ = ants.NewPool(int(common.SignVerifyCore))
	return &p
}

func (tp *TxPool) AddTx(txs []*block.Transaction) {
	if common.SignatureVerify {
		succ, err := consensus.VerifyTxsEcdsaSigMul(txs, tp.sigPool)
		if !succ {
			log.Error("verify fail", "err", err)
			return
		}
	} else {
		// 不验证签名
	}

	tp.lock.Lock()
	defer tp.lock.Unlock()

	tp.pending = append(tp.pending, txs...)
}

func (tp *TxPool) ReadTx(parent, hash common.Hash, num int) []*block.Transaction {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	if tp.pending == nil || len(tp.pending) == 0 {
		return make([]*block.Transaction, 0)
	}
	v, ok := tp.parentNonce[parent]
	if !ok {
		v = 0
	}
	//往前取1
	start := v + 1
	if len(tp.pending) == 0 {
		tp.parentNonce[hash] = v
		return make([]*block.Transaction, 0)
	}
	c := uint64(len(tp.pending))
	if tp.pending != nil && c > start {
		end := start + uint64(num) + 1
		if c > end {
			tp.parentNonce[hash] = end
			//log.Info("parentnonce", "nonce", txs[end-1].Nonce)
			return tp.pending[start:end]
		} else {
			tp.parentNonce[hash] = c
			return tp.pending[start:]
		}
	}
	tp.parentNonce[hash] = v
	return make([]*block.Transaction, 0)
}
