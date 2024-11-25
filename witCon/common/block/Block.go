package block

import (
	"io"
	"time"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/log"
)

type Block struct {
	Number     uint64      //高度
	ParentHash common.Hash //父hash
	LedgerHash common.Hash //
	Payload    []byte      //额外的字段
	Coinbase   common.Address
	TimeStamp  uint64

	Txs   Transactions
	Hash  common.Hash //Hash值
	Extra []byte

	Proof     []byte
	ShardBody []*ShardBody
}

func NewBlock(number uint64, parentHash common.Hash, extra []byte, Coinbase common.Address) *Block {
	b := newBlock(number, parentHash, extra, Coinbase)
	return b
}

func newBlock(number uint64, parentHash common.Hash, extra []byte, Coinbase common.Address) *Block {
	b := &Block{
		Number:     number,
		ParentHash: parentHash,
		Coinbase:   Coinbase,
		Extra:      extra,
	}
	//b.Payload = common.GetPayload()
	if number == 0 {
		b.TimeStamp = 0
	} else {
		b.TimeStamp = uint64(time.Now().UnixMilli())
	}
	if b.Payload == nil {
		b.Payload = []byte{}
	}
	b.Hash = b.RlpHash()
	return b
}

func (b *Block) SetLedgerHash(ledgerHash common.Hash) {
	b.LedgerHash = ledgerHash
}

//func (b *Block) Signature(prv *ecdsa.PrivateKey) {
//	b.Sig, _ = crypto.Sign(b.Hash.Bytes(), prv)
//}

func (b *Block) RlpHash() (h common.Hash) {
	h = crypto.EncodeHash(func(writer io.Writer) {
		rlp.Encode(writer, []interface{}{
			b.Number,
			b.ParentHash,
			b.Coinbase,
			b.TimeStamp,
			b.Payload,
			b.Extra,
		})
	})
	return h
}

func (b *Block) View() uint64 {
	return common.BytesToUint64(b.Extra)
}

func (b *Block) SetTxs(txs []*Transaction) {
	b.Txs = txs
}

func (b *Block) QC() *Vote {
	if b.Number == 0 {
		return &Vote{
			User:   common.Address{},
			Number: 0,
			Status: 0,
			BC:     common.Hash{},
			Sig:    nil,
		}
	}
	return VoteFromByte(b.Extra)
}

func (b *Block) JolteonView() uint64 {
	return common.BytesToUint64(b.Extra[:8])
}

func (b *Block) JolteonQC() *Vote {
	if b.Number == 0 {
		return &Vote{
			User:   common.Address{},
			Number: 0,
			Status: 0,
			BC:     common.Hash{},
			Sig:    nil,
		}
	}
	return VoteFromByte(b.Extra[8:])
}
func (b *Block) ShallowCopyBC() *Block {
	return &Block{
		Number:     b.Number,
		ParentHash: b.ParentHash,
		LedgerHash: b.LedgerHash,
		Payload:    b.Payload,
		Coinbase:   b.Coinbase,
		TimeStamp:  b.TimeStamp,
		Txs:        b.Txs,
		Hash:       b.Hash,
		Extra:      b.Extra,
		Proof:      b.Proof,
		ShardBody:  nil,
	}
}

func (b *Block) ShallowCopyShard(shard []uint) *Block {
	_shardBody := make([]*ShardBody, len(shard))
	if len(b.ShardBody) == 0 {
		log.Crit("shard body nil")
	}
	for i, key := range shard {
		_shardBody[i] = b.ShardBody[key]
	}
	return &Block{
		Number:     b.Number,
		ParentHash: b.ParentHash,
		LedgerHash: b.LedgerHash,
		Payload:    b.Payload,
		Coinbase:   b.Coinbase,
		TimeStamp:  b.TimeStamp,
		Txs:        nil,
		Hash:       b.Hash,
		Extra:      b.Extra,
		Proof:      b.Proof,
		ShardBody:  _shardBody,
	}
}
