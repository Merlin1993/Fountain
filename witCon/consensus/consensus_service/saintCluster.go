package consensus_service

import "witCon/common"

type SaintCluster interface {
	Coinbase() common.Address
	SaintLen() int
	Turn() uint
	Tolerance() int
	GetSaint(offset int) common.Address
	GetSaintTurn(addr common.Address) int
	Rotation() bool
}
