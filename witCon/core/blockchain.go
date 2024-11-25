package core

import (
	lru "github.com/hashicorp/golang-lru"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	rawbd "witCon/db"
	"witCon/log"
	"witCon/stat"
)

type blockchain struct {
	blockCache *lru.ARCCache
	genesis    *block.Block
	db         rawbd.Database
	lastWrite  uint64
	writeCh    chan *block.Block
}

func NewBlockchain(db rawbd.Database) *blockchain {
	bc := &blockchain{
		db:        db,
		lastWrite: 0,
		writeCh:   make(chan *block.Block, 10),
	}
	bc.blockCache, _ = lru.NewARC(1000)
	log.NewGoroutine(bc.writeLoop)
	return bc
}

func (p *blockchain) writeLoop() {
	for bc := range p.writeCh {
		p.writeBlock(bc)
	}
}

func (p *blockchain) SetGenesis(genesis *block.Block) {
	p.genesis = genesis
}

func (p *blockchain) GetBlock(hash common.Hash) *block.Block {
	bc2, ok := p.blockCache.Get(hash)
	if ok {
		return bc2.(*block.Block)
	}
	bcV, err := p.db.Get(hash.Bytes())
	if err == nil && len(bcV) > 0 {
		bc3 := &block.Block{}
		err = rlp.DecodeBytes(bcV, bc3)
		if err != nil {
			log.Error("getblock decode", "err", err)
		}
		p.blockCache.Add(bc3.Hash, bc3)
		return bc3
	}
	if hash == p.genesis.Hash {
		return p.genesis
	}
	return nil
}

func (p *blockchain) WriteBlock(bc *block.Block) {
	p.blockCache.Add(bc.Hash, bc)
	p.writeCh <- bc
	//p.writeBlock(bc)
}

func (p *blockchain) writeBlock(bc *block.Block) {
	if bc.Number != p.lastWrite+1 && bc.Number != 0 {
		log.Crit("write fail!!", "number", bc.Number, "last", p.lastWrite)
	}
	p.lastWrite = bc.Number
	bc.Proof = []byte{1}
	bytes, err := rlp.EncodeToBytes(bc)
	if err != nil {
		log.Error("writeBlock", "err", err)
	}
	stat.Instance.OnBlockWrite(bc.Hash, len(bytes))
	if len(common.VerifyNode) > 0 {
		stat.Instance.DoLock(func() {
			for _, sb := range bc.ShardBody {
				stat.Instance.OnCTxWrite(len(sb.Txs))
			}
		})
	} else {
		if bc.Txs != nil {
			stat.Instance.DoLock(func() {
				for _, tx := range bc.Txs {
					stat.Instance.OnTxWrite(tx.TxHash)
				}
			})
		}
	}
	err = p.db.Put(bc.Hash.Bytes(), bytes)
	if err != nil {
		log.Error("writeBlock put db", "err", err)
	}
}
