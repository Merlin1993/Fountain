package block

import (
	"io"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/log"
)

//rlp编码对空字段存储占用表现良好，所以不用担心存储问题。
type Vote struct {
	User        common.Address //投票人
	Number      uint64
	Status      uint //进行的回合
	ConfirmHash common.Hash
	BC          common.Hash //选择的区块
	Sig         []byte
}

func (v *Vote) SetSig(sig []byte) {
	v.Sig = sig
}

func (v *Vote) RlpHash() (h common.Hash) {
	h = crypto.EncodeHash(func(writer io.Writer) {
		rlp.Encode(writer, []interface{}{
			v.User,
			v.Number,
			v.Status,
			v.BC,
		})
	})
	return h
}

func (v *Vote) ToByte() []byte {
	b, err := rlp.EncodeToBytes(v)
	if err != nil {
		log.Error("encode vote fail", "err", err)
		return nil
	}
	return b
}

func VoteFromByte(b []byte) *Vote {
	var v = &Vote{}
	err := rlp.DecodeBytes(b, v)
	if err != nil {
		log.Error("decode vote fail", "err", err)
		return nil
	}
	return v
}
