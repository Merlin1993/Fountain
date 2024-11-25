package core

import "witCon/common"

//用于确定有权限进行共识的节点
type SaintClusterImpl struct {
	author    common.Address
	saintList []common.Address
	turn      uint
	saintLen  int
	f2        int //确认的人数

	rotation bool
}

func NewNodeCluster(author common.Address, saintList []common.Address, rotation bool) *SaintClusterImpl {
	cfg := &SaintClusterImpl{
		author:    author,
		saintList: saintList,
		saintLen:  len(saintList),
		rotation:  rotation,
	}
	cfg.f2 = cfg.saintLen * 2 / 3
	for index, addr := range saintList {
		if addr == author {
			cfg.turn = uint(index)
			break
		}
	}
	return cfg
}

func (nci *SaintClusterImpl) Coinbase() common.Address {
	return nci.author
}

func (nci *SaintClusterImpl) SaintLen() int {
	return nci.saintLen
}

func (nci *SaintClusterImpl) Turn() uint {
	return nci.turn
}

func (nci *SaintClusterImpl) Tolerance() int {
	return nci.f2
}

func (nci *SaintClusterImpl) GetSaint(offset int) common.Address {
	return nci.saintList[offset]
}

func (nci *SaintClusterImpl) GetSaintTurn(addr common.Address) int {
	for index, address := range nci.saintList {
		if address == addr {
			return index
		}
	}
	return 0
}

func (nci *SaintClusterImpl) Rotation() bool {
	return nci.rotation
}
