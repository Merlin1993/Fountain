package block

import (
	"io"
	"math/big"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/crypto"
)

type Transaction struct {
	Number uint64
	From   common.Address
	Data   []byte
	Sig    []byte

	TxHash common.Hash
}

func NewTx(from common.Address, to common.Address, amount *big.Int, number uint64) *Transaction {
	data := append([]byte{0, 0, 0, 0, 0, 0, 0, 2}, to.Bytes()...)
	data = append(data, amount.Bytes()...)
	tx := &Transaction{
		From:   from,
		Data:   data,
		Number: number,
	}
	tx.TxHash = tx.RlpHash()
	return tx
}

func (tx *Transaction) SetSig(pk [33]byte, sig []byte) {
	tx.Sig = pk[:]
	tx.Sig = append(tx.Sig, sig[:]...)
}

func (tx *Transaction) Sign(sig []byte) {
	tx.Sig = sig
}

func (tx *Transaction) RlpHash() (h common.Hash) {
	h = crypto.EncodeHash(func(writer io.Writer) {
		rlp.Encode(writer, []interface{}{
			tx.Number,
			tx.From,
			tx.Data,
		})
	})
	return h
}

type Transactions []*Transaction

//func (txs Transactions) ToHashes()common.Hashes{
//	hashes := make(common.Hashes txs)
//}
