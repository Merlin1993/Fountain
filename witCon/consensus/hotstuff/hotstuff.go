package hotstuff

import (
	"math"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/consensus/consensus_service"
	"witCon/log"
)

const (
	NextSyncView = iota
	SendVoteCode
)

type Hotstuff struct {
	consensus_service.ConsensusProxy
	consensus_service.Seal
	//consensus.ChainRW
	//*consensus.ProtoHandler
	cfg       consensus_service.SaintCluster
	vHeight   uint64
	genesis   *block.Block
	BlockLock *block.Block
	BlockExec *block.Block
	QCHigh    *block.QC
	BlockLeaf *block.Block
	BlockTail *block.Block
	blockPool map[common.Hash]*block.Block

	sender     consensus_service.AdditionSender
	nextViewCh chan struct{}

	votes map[common.Hash]map[common.Address]*block.Vote
	//pendingTask map[common.Hash]func()
	timer *time.Timer
}

func NewHotstuff(nc consensus_service.SaintCluster, sender consensus_service.AdditionSender,
	genesis *block.Block, seal consensus_service.Seal, proxy consensus_service.ConsensusProxy) *Hotstuff {
	h := &Hotstuff{
		//ChainRW:      bc,
		//ProtoHandler: consensus.newProtoHandle(nc.SaintLen()),
		cfg:       nc,
		Seal:      seal,
		sender:    sender,
		vHeight:   0,
		blockPool: make(map[common.Hash]*block.Block),
		votes:     make(map[common.Hash]map[common.Address]*block.Vote),
		//pendingTask:  make(map[common.Hash]func()),
		nextViewCh:     make(chan struct{}),
		ConsensusProxy: proxy,
	}
	h.genesis = genesis
	h.BlockLock = h.genesis
	h.BlockExec = h.genesis
	h.QCHigh = &block.QC{
		Block: h.genesis,
		Justify: &block.Vote{
			User:   nc.Coinbase(),
			Number: 0,
			Status: 0,
			BC:     h.genesis.Hash,
		},
	}
	h.blockPool[h.genesis.Hash] = h.genesis
	h.BlockLeaf = h.genesis
	h.BlockTail = h.genesis
	//go h.mainLoop()
	go h.viewchangeLoop()
	return h
}

func (h *Hotstuff) CommitSignature(v *block.Vote) {
	//啥也不用做，因为hotstuff是直接发给区块所有者的，所以不会由上层proxy转发
	//h.onReceiveVote(v)
}

func (h *Hotstuff) getBlock(hash common.Hash) *block.Block {
	bc, ok := h.blockPool[hash]
	if ok {
		return bc
	}
	if h.BlockExec.Hash == hash {
		return h.BlockExec
	}
	return nil
}

func (h *Hotstuff) ExistBlock(hash common.Hash) bool {
	_, ok := h.blockPool[hash]
	return ok
}
func (h *Hotstuff) GetProcessBlock(hash common.Hash) (interface{}, bool) {
	bc, ok := h.blockPool[hash]
	return bc, ok
}
func (h *Hotstuff) CommitBlock(bc *block.Block, v []*block.Vote) {
	if ok, ancenstBlock := h.verifyQC(bc.QC()); ok {
		//如果不是父区块，也可能是最高的qc，直接试试咯
		if bc.ParentHash != ancenstBlock {
			pb := h.getBlock(bc.ParentHash)
			ab := h.getBlock(ancenstBlock)
			if pb == nil || ab == nil {
				log.Error("verifyQC ancenstBlock fail", "pb", pb, "ab", ab)
				return
			}
			if !h.extends(pb, ab) {
				log.Error("verifyQC fail, not ancent block", "pb", pb.Hash, "number", pb.Number, "parent", pb.ParentHash, "ab", ab.Hash, "ab number", ab.Number)
				return
			} else {
				log.Debug("verifyQC, ancent block")
			}
		}
	} else {
		log.Error("verifyQC fail")
		return
	}
	h.blockPool[bc.Hash] = bc
	h.onReceiveBlock(bc)
}

func (h *Hotstuff) verifyQC(v *block.Vote) (ok bool, ancestBlock common.Hash) {
	if v.Number == 0 {
		return true, h.genesis.Hash
	}
	var vs = make([]*block.Vote, 0)
	err := rlp.DecodeBytes(v.Sig, &vs)
	if err != nil {
		log.Error("verifyQC", "err", err)
		return false, common.Hash{}
	}
	vm := make(map[common.Address]struct{})
	QCB := vs[0].BC
	a := make(chan struct{}, len(vs))
	fail := make(chan struct{})
	for _, v := range vs {
		vm[v.User] = struct{}{}
		if v.BC != QCB {
			log.Error("verifyQC QCB fail", "v", v.BC, "qcb", QCB)
			return false, common.Hash{}
		}
		vt := v
		go func() {
			if !h.Seal.Verify(vt.Sig, vt.RlpHash(), vt.User) {
				log.Error("verifyQC sig fail", "v", vt)
				fail <- struct{}{}
			} else {
				a <- struct{}{}
			}
		}()
	}
	if len(vm) < h.cfg.Tolerance() {
		log.Error("verifyQC  len fail", "len", len(vm))
		return false, common.Hash{}
	}
	count := 0
	for {
		select {
		case <-a:
			count++
			if count == len(vs) {
				return true, QCB
			}
		case <-fail:
			return false, QCB
		}
	}
}

func (h *Hotstuff) StartDaemon(daemon common.Address) {
	log.Info("action")
	//go h.onBeat()
}

func (h *Hotstuff) ExtraInfo() []byte {
	return h.QCHigh.Justify.ToByte()
}

func (h *Hotstuff) CheckPackAuth(num uint64) bool {
	leader := h.getLeader()
	log.Debug("CheckPackAuth", "leader", leader, "number", h.BlockTail.Number)
	return leader == h.cfg.Coinbase()
}

func (h *Hotstuff) HandleMsg(addr common.Address, code uint, data []byte) {
	switch code {
	case NextSyncView:
		vote := &block.QC{}
		err := rlp.DecodeBytes(data, vote)
		log.Debug("rec NextSyncView", "vote", vote)
		if err != nil {
			log.Error("rec NextSyncView", "err", err)
		}
		h.onReceiveNewView(vote)
	case SendVoteCode:
		v := &block.Vote{}
		err := rlp.DecodeBytes(data, v)
		log.Debug("rec SendVoteCode", "vote", v.BC, "user", v.User, "num", v.Number, "status", v.Status)
		if err != nil {
			log.Error("rec SendVoteCode", "err", err)
		}
		if h.Verify(v.Sig, v.RlpHash(), v.User) {
			h.onReceiveVote(v)
		} else {
			log.Error("rec err vote", "v", v)
		}
	}
}

//检查区块的引用
//func (h *Hotstuff) mainLoop() {
//	for {
//		select {
//		case bc := <-h.ProtoHandler.minerCh:
//			h.blockPool[bc.Hash] = bc
//			//if h.CheckBlock(bc.Coinbase, bc.ParentHash, func() { h.pendingTask[bc.Hash]() }) {
//			//	log.Debug("requestblock by send")
//			//	return
//			//}
//			//if h.CheckBlock(bc.Coinbase, bc.QCHash, func() { h.pendingTask[bc.Hash]() }) {
//			//	log.Debug("requestblock by send qc hash")
//			//	return
//			//}
//
//			h.checkBlock(bc.Coinbase, bc.ParentHash)
//			h.checkBlock(bc.Coinbase, bc.QCHash)
//			if h.extends(h.BlockExec, bc) {
//				h.onCommit(bc)
//			}
//			//if fn, ok := h.pendingTask[bc.Hash]; ok {
//			//	fn()
//			//	delete(h.pendingTask, bc.Hash)
//			//}
//		case <-h.nextViewCh:
//
//		}
//	}
//}

func (h *Hotstuff) viewchangeLoop() {
	h.timer = time.NewTimer(0)
	<-h.timer.C
	for {
		select {
		case <-h.timer.C:
			h.SynchronizeRun(func() {
				h.onNextSyncView()
			})
		}
	}
}

//func (h *Hotstuff) checkBlock(name common.Address, hash common.Hash) {
//	if h.GetBlock(hash) == nil {
//		h.ProtoHandler.RequestBlock(name, hash)
//	}
//}
//
//func (h *Hotstuff) CheckBlock(name common.Address, hash common.Hash, fn func()) bool {
//	if h.GetBlock(hash) == nil {
//		h.ProtoHandler.RequestBlock(name, hash)
//		lastTask, ok := h.pendingTask[hash]
//		h.pendingTask[hash] = func() {
//			if ok {
//				lastTask()
//			}
//			fn()
//		}
//		return true
//	}
//	return false
//}

func (h *Hotstuff) update(bc *block.Block) {
	log.Debug("update", "number", bc.Number, "qcHeight", bc.QC().Number, "parent", bc.ParentHash, "genesis", h.genesis.Hash)
	prepareBc := h.getBlock(bc.QC().BC)
	if prepareBc == nil {
		log.Error("prepareBc nil!!", "number", bc.Number, "qcHeight", bc.QC().Number, "parent", bc.ParentHash, "genesis", h.genesis.Hash)
		return
	}
	h.updateQcHigh(&block.QC{
		Block:   prepareBc,
		Justify: bc.QC(),
	})
	preCommitBc := h.getBlock(prepareBc.QC().BC)
	//一般区块高度比较低（第一个第二个区块），就会触发空
	if preCommitBc == nil {
		return
	}
	if preCommitBc.Number > h.BlockLock.Number {
		h.BlockLock = preCommitBc
	}
	commitBc := h.getBlock(preCommitBc.QC().BC)
	if commitBc == nil {
		return
	}
	if prepareBc.ParentHash == preCommitBc.Hash && preCommitBc.ParentHash == commitBc.Hash {
		h.onCommit(commitBc)
		h.BlockExec = commitBc
	}
}

func (h *Hotstuff) updateQcHigh(v *block.QC) {
	log.Debug("updateQcHigh", "block", v.Block.Number, "qcHeight", v.Justify.Number, "local QC", h.QCHigh.Block.Number)
	//在这里，hotstuff允许了未收集满见证的区块继续后面出块，但是只能引用最高的QC
	if v.Block.Number > h.QCHigh.Block.Number {
		h.QCHigh = v
		h.BlockLeaf = v.Block
	}
}

func (h *Hotstuff) onCommit(bc *block.Block) {
	if h.BlockExec.Number < bc.Number {
		bc2 := h.getBlock(bc.ParentHash)
		//如果没有父区块，那么就等一等
		if bc2 == nil {
			log.Error("onCommit block parent nil", "bc", bc.Hash, "number", bc.Number)
			return
		}
		h.onCommit(bc2)
		log.Debug("WriteBlock", "bc", bc.Hash, "number", bc.Number)
		//if bc.Number%10000 == 0 {
		//	log.Info("WriteBlock", "bc", bc.Hash, "number", bc.Number)
		//}
		delete(h.blockPool, bc.Hash)
		h.OnBlockConfirm(bc)
	}
}

func (h *Hotstuff) onReceiveBlock(bc *block.Block) {
	if bc.Number <= h.vHeight {
		log.Debug("on Receive low Proposal", "receive", bc.Number, "number", h.vHeight)
		return
	}
	//如果是子孙区块，或者是投票和的高度在lock以后，那么就发送投票
	if h.extends(bc, h.BlockLock) || (bc.QC().Number > h.BlockLock.Number || bc.QC().Number == 0) {
		h.vHeight = bc.Number
		h.BlockTail = bc
		h.OnBlockAvailable(h.BlockTail)
		leader := h.getLeader()
		log.Debug("SendVote", "leader", leader, "hash", bc.Hash, "number", bc.Number)
		vote := &block.Vote{
			User:   h.cfg.Coinbase(),
			Number: bc.Number,
			Status: 0,
			BC:     bc.Hash,
		}
		sig, _ := h.Signature(vote.RlpHash())
		vote.SetSig(sig)
		if leader == h.cfg.Coinbase() {
			h.update(bc)
			h.onReceiveVote(vote)
		} else {
			h.SendVote(leader, vote)
			h.update(bc)
		}
	} else {
		//更新
		h.update(bc)
	}
	//h.checkVoteEnough(bc.Hash) //不需要吧，因为不会的
}

//此处没有限定
//我们采取上一个区块接受的时间，到一个间隔时间，间隔时间T内我们选第一个，T-3T内我们选第二个，3T-7T内我们选第三个
func (h *Hotstuff) getLeader() common.Address {
	if h.BlockTail.Hash == h.genesis.Hash {
		return h.cfg.GetSaint(0)
	}
	//取对数
	oldOffset := h.cfg.GetSaintTurn(h.BlockTail.Coinbase)
	if oldOffset != 0 {
		//log.Error("old offset is not 0")
	}
	ti := (time.Now().UnixMilli() - int64(h.BlockTail.TimeStamp)) / consensus_service.ViewChangeDuration.Milliseconds()
	offset := math.Ceil(math.Log2(float64(ti))) + 1
	turn := uint64(0)
	if h.cfg.Rotation() {
		turn = (h.BlockTail.Number + 1) % uint64(h.cfg.SaintLen())
	}
	offsetTurn := (turn + uint64(offset) + uint64(oldOffset)) % uint64(h.cfg.SaintLen())
	return h.cfg.GetSaint(int(offsetTurn))
}

func (h *Hotstuff) extends(son, father *block.Block) bool {
	for son.Number > father.Number {
		if son.ParentHash == father.Hash {
			return true
		}
		son = h.getBlock(son.ParentHash)
		if son == nil {
			return false
		}
	}
	return son.Hash == father.Hash
}

func (h *Hotstuff) onReceiveVote(vote *block.Vote) {
	log.Debug("onReceiveVote", "vote_num", vote.Number, "vote_user", vote.User)
	if h.QCHigh.Justify.Number == vote.Number {
		log.Debug("onReceiveVote", "justify", h.QCHigh.Justify.Number, "vote_num", vote.Number)
		return
	}
	vs, ok := h.votes[vote.BC]
	if !ok {
		vs = make(map[common.Address]*block.Vote)
		h.votes[vote.BC] = vs
	}
	vs[vote.User] = vote
	h.checkVoteEnough(vote.BC)
}

func (h *Hotstuff) checkVoteEnough(hash common.Hash) {
	vs, ok := h.votes[hash]
	if !ok {
		return
	}
	bc := h.getBlock(hash)
	if bc == nil {
		log.Debug("onReceiveVote block is nil", "bc", hash)
		return
	}
	//达到阈值
	if len(vs) > h.cfg.Tolerance() {
		log.Debug("receive enough vote", "votes", vs, "len", len(vs), "num", bc.Number,
			"getLeader", h.getLeader(), "self", h.cfg.Coinbase().String())
		group := h.cfg.Coinbase()
		voteList := make([]*block.Vote, len(vs))
		i := 0
		for _, v := range vs {
			voteList[i] = v
			i++
		}
		vote := &block.Vote{
			User:   group,
			Number: bc.Number,
			Status: 0,
			BC:     bc.Hash,
		}
		b, err := rlp.EncodeToBytes(voteList)
		if err != nil {
			log.Error("encode voteList fail", "err", err)
		}
		vote.SetSig(b)
		qc := &block.QC{
			Block:   bc,
			Justify: vote,
		}
		h.updateQcHigh(qc)
		if qc.Justify.Number == h.BlockTail.Number {
			h.onBeat()
		}
	}
}

//
//func (h *Hotstuff) onPropose(bc *block.Block, qc *block.QC) *block.Block {
//	blockNew := h.createLeaf(bc, qc.Justify, bc.Number+1)
//	proposal := &block.Block{
//		Block:  blockNew,
//		LastQc: qc.Justify,
//	}
//	h.BlockTail = blockNew
//	h.blockPool[proposal.Hash] = proposal
//	h.BroadcastProposal(proposal)
//	//也要让自己收到
//	return proposal
//}

func (h *Hotstuff) onBeat() {
	//是当前的节点，就发送提案，然后计时
	leader := h.getLeader()
	log.Debug("onBeat", "leader", leader, "number", h.BlockTail.Number)
	if leader == h.cfg.Coinbase() {
		//log.Info("onBeat", "leader", leader, "number", h.BlockTail.Number)
		h.DoPackBlock()

		//h.BlockTail = blockNew
		//h.blockPool[proposal.Hash] = proposal

		//h.onReceiveBlock(bc)
		//h.onReceiveVote(&block.Vote{
		//	User:   h.cfg.Coinbase(),
		//	Number: bc.Number,
		//	Status: 0,
		//	BC:     bc.Hash,
		//})
	}
	h.timer.Reset(consensus_service.ViewChangeDuration)
}

func (h *Hotstuff) onNextSyncView() {
	leader := h.getLeader()
	if leader == h.cfg.Coinbase() {
		//log.Error("next is myself")
		h.onBeat()
	} else {
		//log.Error("next is others")
		h.SendNextSyncView(leader, h.QCHigh)
		h.timer.Reset(consensus_service.ViewChangeDuration)
	}
}

func (ps *Hotstuff) SendNextSyncView(name common.Address, vote *block.QC) {
	ps.sender.AdditionSend(name, NextSyncView, vote)
}

func (ps *Hotstuff) SendVote(name common.Address, v *block.Vote) {
	ps.sender.AdditionSend(name, SendVoteCode, v)
}

func (h *Hotstuff) onReceiveNewView(qc *block.QC) {
	h.updateQcHigh(qc)
}
