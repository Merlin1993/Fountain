package consensus_service

import (
	"github.com/panjf2000/ants/v2"
	"witCon/common"
	"witCon/common/block"
)

type BaseConsensusImpl interface {
	CommitSignature(v *block.Vote)
	ExistBlock(hash common.Hash) bool //必须检查所有未确定的区块
	GetProcessBlock(hash common.Hash) (interface{}, bool)
}

type ConsensusImpl interface {
	BaseConsensusImpl
	CommitBlock(bc *block.Block, v []*block.Vote)
	StartDaemon(daemon common.Address)
	ExtraInfo() []byte
	CheckPackAuth(num uint64) bool
}

type ConsensusProxy interface {
	//OnBranchExist(ProofBlock)         //区块出现分岔
	OnBlockConfirm(bc *block.Block)    //区块已被确认
	OnBlockCFTConfirm(bc *block.Block) //区块已经可以被引用
	OnBlockAvailable(bc *block.Block)  //区块已经可以被引用
	OnVote(v *block.Vote)
	//OnBlockProcess(ProcessBlock) //区块已经开始共识
	SynchronizeRun(task func()) //执行需要同步的操作，避免go程形成异步

	//PBFT所需
	ResendCurrentBlock(addr common.Address)
	DoPackBlock()
}

// 共识协议特殊的协议
type AdditionHandler interface {
	HandleMsg(addr common.Address, code uint, data []byte)
}

type AdditionSender interface {
	AdditionSend(addr common.Address, code uint64, data interface{})
	AdditionBroadcast(code uint, data interface{})
}

type TxPool interface {
	ReadTx(parent, hash common.Hash, num int) []*block.Transaction
	StartTxRate(num int)
}

type WorldState interface {
	ExecuteBc(bc *block.Block, txs []*block.Transaction) ([]*block.ShardBody, common.Hash, error)
	VerifyShardMulti(bc *block.Block, body []*block.ShardBody, pool *ants.Pool) error
	VerifyShard(bc *block.Block, body *block.ShardBody, shardIndex uint16) error
	OnBlockConfirm(bc *block.Block)
}

//	type WitConsensus interface {
//		CommitSignature(signature *block.Signature)
//		ExistBlock(hash common.Hash) bool //必须检查所有未确定的区块
//		GetProcessBlock(hash common.Hash) (interface{}, bool)
//	}
//
//	type DaemonWitConsensus interface {
//		WitConsensus
//		CommitBlock(dblock *block.DBlock, signature []*block.Signature)
//		StartDaemon(daemon common.Address)
//	}
//
//	type ProcessBlock interface {
//		Block() interface{}
//		Hash() common.Hash
//	}
//
//	type ProofBlock interface {
//		GenerateProof() *block.WitnessProof
//		Block() interface{}
//		BlockMap() []interface{}
//		NumberAt() *big.Int
//		GetBlock(hash common.Hash) interface{}
//	}
//
//	type ConsensusSM interface {
//		OnBranchExist(ProofBlock)      //区块出现分岔
//		OnBlockConfirm(ProofBlock)     //区块已被确认
//		OnBlockAvailable(ProcessBlock) //区块已经可以被引用
//		OnBlockProcess(ProcessBlock)   //区块已经开始共识
//		SynchronizeRun(task func())    //执行需要同步的操作，避免go程形成异步
//	}
//
//	type NodeCluster interface {
//		ConsensusNode(hash common.Hash) ([]common.Address, error)
//	}
type Seal interface {
	Signature(hash common.Hash) (sign []byte, error error)
	Coinbase() common.Address
	Verify(sig []byte, hash common.Hash, addr common.Address) bool
}
