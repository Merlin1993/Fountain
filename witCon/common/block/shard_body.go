package block

import (
	"encoding/json"
	"fmt"
	"math/big"
	"witCon/common"
	"witCon/common/hexutil"
	"witCon/crypto"
	"witCon/log"
)

type ShardBody struct {
	ReadState  [][]byte
	ShardProof []*crypto.MerkleTreeProof
	MultiProof *crypto.MultiMerkleProof
	SelfOp     []*StateOp
	Txs        []*Transaction
	TxIndex    []uint16
}

func (sb *ShardBody) Print() {
	if !common.PrintMerkleTree {
		return
	}
	log.Debug("-------------state----------")
	//for index, rs := range sb.ReadState {
	//	if rs != nil {
	//		log.Debug("state", "index", index, "state", hexutil.Encode(rs), "value", big.NewInt(0).SetBytes(rs))
	//	} else {
	//		log.Debug("state", "index", index, "state", "nil")
	//	}
	//}
	for index, sp := range sb.ShardProof {
		log.Debug("proof", "index", index)
		if sp != nil {
			sp.Print()
		} else {
			log.Debug("proof is nil")
		}
	}
	//log.Debug(fmt.Sprintf("%v", sb.SelfOp))
	//log.Debug(fmt.Sprintf("%v", sb.Txs))
	//log.Debug(fmt.Sprintf("%v", sb.TxIndex))
}

type StateOp struct {
	Index   uint16
	InShard uint16
	Read    bool
	Key     common.Address
	Value   []byte
}

func (so *StateOp) String() string {
	j, err := json.Marshal(so)
	if err != nil {
		log.Error("stateop json fail", "err", err)
		return ""
	}
	return string(j) + fmt.Sprintf("ps:value:%v;%v", big.NewInt(0).SetBytes(so.Value).String(), hexutil.Encode(so.Value))
}
