package chaincode

import "witCon/common"

type VM interface {
	ReadState(key common.Address) []byte
	WriteState(key common.Address, value []byte)
}
