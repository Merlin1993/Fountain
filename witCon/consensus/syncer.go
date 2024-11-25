package consensus

import (
	lru "github.com/hashicorp/golang-lru"
	"witCon/common"
	"witCon/common/block"
	"witCon/log"
)

type Syncer interface {
	//暂存投票
	CheckVoteExist(*block.Vote) bool

	//区块同步
	CheckBlockRef(common.Address, *block.Block)
	AddPendingBlock(bc *block.Block)

	GetTempVote(hash common.Hash) []*block.Vote

	//区块同步完提示提示
	OnWriteBlock(*block.Block)
}

type SyncerEngine interface {
	CommitBlock(block *block.Block)
	VerifyBlock(block *block.Block) bool
	RequestBlock(name common.Address, hash common.Hash)
	HasAvailableBlock(hash common.Hash, num uint64) bool //存在可用可被引用的区块
	HasBlock(hash common.Hash) bool                      //存在这个区块
}

type SyncerImpl struct {
	SyncerEngine
	crw          ChainRW
	taskChan     chan func()   //待分配任务的通道
	pendingBlock *lru.ARCCache //等待同步的区块
	waitingBlock *lru.ARCCache //等待同步的区块
	tempVote     *lru.ARCCache //暂存的投票
}

func NewSyncerImpl(engine SyncerEngine, chain ChainRW) *SyncerImpl {
	si := &SyncerImpl{
		SyncerEngine: engine,
		taskChan:     make(chan func(), 1000),
		crw:          chain,
	}
	si.pendingBlock, _ = lru.NewARC(100000)
	si.waitingBlock, _ = lru.NewARC(1000)
	si.tempVote, _ = lru.NewARC(1000)
	log.NewGoroutine(si.preBlockLoop)
	return si
}

func (si *SyncerImpl) preBlockLoop() {
	for {
		select {
		case task := <-si.taskChan:
			task()
		}
	}
}

func (si *SyncerImpl) CheckVoteExist(v *block.Vote) bool {
	if si.HasBlock(v.BC) {
		return true
	}
	si.addTempSignature(v)
	return false
}

func (si *SyncerImpl) addTempSignature(v *block.Vote) {
	var signatures []*block.Vote
	value, ok := si.tempVote.Get(v.BC)
	if ok {
		signatures = value.([]*block.Vote)
	} else {
		signatures = make([]*block.Vote, 0)
	}
	for _, item := range signatures {
		if item.User == v.User && item.Status == v.Status {
			return
		}
	}
	signatures = append(signatures, v)
	si.tempVote.Add(v.BC, signatures)
}

func (si *SyncerImpl) GetTempVote(hash common.Hash) []*block.Vote {
	var signatures []*block.Vote
	value, ok := si.tempVote.Get(hash)
	if ok {
		signatures = value.([]*block.Vote)
		si.tempVote.Remove(hash)
	} else {
		signatures = make([]*block.Vote, 0)
	}
	return signatures
}

func (si *SyncerImpl) AddPendingBlock(bc *block.Block) {
	si.pendingBlock.Add(bc.Number, bc)
}

func (si *SyncerImpl) CheckBlockRef(res common.Address, bc *block.Block) {
	si.taskChan <- func() {
		if si.HasAvailableBlock(bc.ParentHash, bc.Number-1) {
			if si.VerifyBlock(bc) {
				si.pendingBlock.Add(bc.Number, bc)
				si.CommitBlock(bc)
			}
			return
		}
		if c, ok := si.pendingBlock.Get(bc.Number - 1); ok {
			if c.(*block.Block).Hash == bc.ParentHash {
				if si.VerifyBlock(bc) {
					if si.HasAvailableBlock(bc.ParentHash, bc.Number-1) {
						si.pendingBlock.Add(bc.Number, bc)
						si.CommitBlock(bc)
						return
					} else {
						si.pendingBlock.Add(bc.Number, bc)
					}
				}
				return
			}
		}
		log.Warn("check block ref fail", "number", bc.Number)
		si.RequestBlock(res, bc.ParentHash)
		si.waitingBlock.Add(bc.ParentHash, bc)
	}
}

func (si *SyncerImpl) OnWriteBlock(bc *block.Block) {
	if v, ok := si.pendingBlock.Get(bc.Number + 1); ok {
		nbc := v.(*block.Block)
		if nbc.ParentHash == bc.Hash {
			si.CommitBlock(nbc)
			return
		}
	}
	if v, ok := si.waitingBlock.Get(bc.Hash); ok {
		if si.VerifyBlock(bc) {
			si.CommitBlock(v.(*block.Block))
		}
	}
}
