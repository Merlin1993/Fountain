package consensus

import (
	"witCon/common"
	"witCon/common/block"
)

type ChainRW interface {
	SetGenesis(genesis *block.Block)
	GetBlock(hash common.Hash) *block.Block
	WriteBlock(bc *block.Block)
}
