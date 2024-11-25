package p2p

import "witCon/common"

type ProtoHandle interface {
	HandleMsg(addr common.Address, code uint, data []byte)
}
