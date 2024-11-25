package core

import (
	"fmt"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/log"
)

type ShardState struct {
	OpList    []*block.StateOp
	StateList []*block.StateOp
}

func (ss *ShardState) String() string {
	return fmt.Sprintf("opList:%v StatrList:%v", ss.OpList, ss.StateList)
}

func (ss *ShardState) AddState(op *block.StateOp) {
	ss.StateList = append(ss.StateList, op)
}

func (ss *ShardState) AddOp(op *block.StateOp) {
	ss.OpList = append(ss.OpList, op)
}

// ReadState  [][]byte
// ShardProof []*MerkleTreeProof
// SelfOp     []*StateOp
// 获取别人写的顺序
func (ss *ShardState) SelfOP(selfShard uint16) []*block.StateOp {
	opl := make([]*block.StateOp, 0, len(ss.StateList)/2)
	for _, op := range ss.StateList {
		if op.InShard != selfShard {
			opl = append(opl, op)
		}
	}
	return opl
}

// 获取自己读的记录
func (ss *ShardState) ReadState(selfShard uint16) [][]byte {
	rsl := make([][]byte, 0, len(ss.OpList)/2)
	for _, op := range ss.OpList {
		//获取所有读别人的状态
		if op.Read && op.InShard == selfShard {
			if op.Value == nil {
				rsl = append(rsl, []byte{})
			} else {
				rsl = append(rsl, op.Value)
			}
		}
	}
	return rsl
}

func (ss *ShardState) ShardOPLen() int {
	return len(ss.StateList)
}

func (ss *ShardState) GetMerkleTree() crypto.MerkleTr {
	return ToCrossShardMerkleRoot([][]*block.StateOp{ss.StateList})
}

// 自己的stateList
func ToCrossShardMerkleRoot(sol [][]*block.StateOp) crypto.MerkleTr {
	crossShardOP := make([][]*block.StateOp, common.InitDirectShardCount)
	//相当于合并多个stateOp列表
	for _, so := range sol {
		for _, sop := range so {
			crossOpList := crossShardOP[sop.InShard]
			if crossOpList == nil {
				crossOpList = make([]*block.StateOp, 1)
				crossOpList[0] = sop
			} else {
				crossOpList = append(crossOpList, sop)
			}
			crossShardOP[sop.InShard] = crossOpList
		}
	}
	if common.CompressMerkle {
		mt := &crypto.CompressMerkleTree{}
		for _, cs := range crossShardOP {
			if cs == nil {
				mt.WriteItem(common.Hash{})
			} else {
				//log.Debug("*****************")
				//log.Debug("cross shard", "index", index)
				//for _, op := range cs {
				//	log.Debug("op", "data", op.String())
				//}
				//log.Debug("*****************")
				data, err := rlp.EncodeToBytes(cs)
				if err != nil {
					log.Crit("fail encode", "err", err)
				}
				mt.WriteItem(crypto.Sha256(data))
			}
		}
		mt.CommitTree()
		return mt
	} else {
		csoHash := make([]common.Hash, common.InitDirectShardCount)
		for index, cs := range crossShardOP {
			if cs == nil {
				csoHash[index] = common.Hash{}
			} else {
				//log.Debug("*****************")
				//log.Debug("cross shard", "index", index)
				//for _, op := range cs {
				//	log.Debug("op", "data", op.String())
				//}
				//log.Debug("*****************")
				data, err := rlp.EncodeToBytes(cs)
				if err != nil {
					log.Crit("fail encode", "err", err)
				}
				csoHash[index] = crypto.Sha256(data)
			}
		}
		mt := &crypto.MerkleTree{}
		mt.MakeTree(csoHash)
		return mt
	}

}
