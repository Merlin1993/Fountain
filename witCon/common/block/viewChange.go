package block

import "witCon/common"

type ViewChange struct {
	LastNum   uint64
	LastProof []*Vote
	View      uint64
	User      common.Address
}

type ViewChangeAck struct {
	LastNum uint64
	View    uint64
	AckUser common.Address
	User    common.Address
}

type NewView struct {
	LastNum uint64
	NewView uint64
	VcLst   []*ViewChange
}

type ViewChangeQC struct {
	LastNum   uint64
	LastProof *QC
	View      uint64
	User      common.Address
}

type ViewChangeQCAck struct {
	LastNum uint64
	View    uint64
	AckUser common.Address
	User    common.Address
}

type NewViewQC struct {
	LastNum uint64
	NewView uint64
	VcLst   []*ViewChangeQC
}
