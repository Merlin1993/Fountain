package consensus

import (
	"witCon/common/block"
)

type Impl interface {
	OnPack() bool
	OnBlockCFTConfirm()
	OnBlockConfirm(bc *block.Block, first bool)
}
