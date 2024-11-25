package consensus

import (
	"witCon/common"
	"witCon/common/block"
)

type SendBlock struct {
	name common.Address
	bc   *block.Block
}
