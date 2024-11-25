package jolteon

import (
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/consensus/consensus_service"
	"witCon/log"
)

type Jolteon struct {
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

	sender consensus_service.AdditionSender

	votes map[common.Hash]map[common.Address]*block.Vote

	vcp *ViewChangeProcess
}

func NewJolteon(nc consensus_service.SaintCluster, sender consensus_service.AdditionSender,
	genesis *block.Block, seal consensus_service.Seal, proxy consensus_service.ConsensusProxy) *Jolteon {
	j := &Jolteon{
		//ChainRW:      bc,
		//ProtoHandler: consensus.newProtoHandle(nc.SaintLen()),
		cfg:            nc,
		Seal:           seal,
		sender:         sender,
		vHeight:        0,
		blockPool:      make(map[common.Hash]*block.Block),
		votes:          make(map[common.Hash]map[common.Address]*block.Vote),
		ConsensusProxy: proxy,
	}
	j.genesis = genesis
	j.BlockLock = j.genesis
	j.BlockExec = j.genesis
	j.QCHigh = &block.QC{
		Block: j.genesis,
		Justify: &block.Vote{
			User:   nc.Coinbase(),
			Number: 0,
			Status: 0,
			BC:     j.genesis.Hash,
		},
	}
	j.blockPool[j.genesis.Hash] = j.genesis
	j.BlockLeaf = j.genesis
	j.BlockTail = j.genesis

	j.vcp = NewViewChangeProcess(nc, sender, j.onNewView)
	return j
}

func (j *Jolteon) CommitSignature(v *block.Vote) {
	//啥也不用做
	//h.onReceiveVote(v)
}

func (j *Jolteon) getBlock(hash common.Hash) *block.Block {
	bc, ok := j.blockPool[hash]
	if ok {
		return bc
	}
	if j.BlockExec.Hash == hash {
		return j.BlockExec
	}
	return nil
}

func (j *Jolteon) ExistBlock(hash common.Hash) bool {
	_, ok := j.blockPool[hash]
	return ok
}
func (j *Jolteon) GetProcessBlock(hash common.Hash) (interface{}, bool) {
	bc, ok := j.blockPool[hash]
	return bc, ok
}
func (j *Jolteon) CommitBlock(bc *block.Block, v []*block.Vote) {
	if ok, ancenstBlock := j.verifyQC(bc.JolteonQC()); ok {
		//如果不是父区块，也可能是最高的qc，直接试试咯
		if bc.ParentHash != ancenstBlock {
			pb := j.getBlock(bc.ParentHash)
			ab := j.getBlock(ancenstBlock)
			if pb == nil || ab == nil {
				log.Error("verifyQC ancenstBlock fail", "pb", pb, "ab", ab)
				return
			}
			if !j.extends(pb, ab) {
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
	j.blockPool[bc.Hash] = bc
	j.onReceiveBlock(bc)
}

func (j *Jolteon) verifyQC(v *block.Vote) (ok bool, ancestBlock common.Hash) {
	if v.Number == 0 {
		return true, j.genesis.Hash
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
			if !j.Seal.Verify(vt.Sig, vt.RlpHash(), vt.User) {
				log.Error("verifyQC sig fail", "v", vt)
				fail <- struct{}{}
			} else {
				a <- struct{}{}
			}
		}()
	}
	if len(vm) < j.cfg.Tolerance() {
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

func (j *Jolteon) StartDaemon(daemon common.Address) {
	log.Info("jolteon action")
	//go h.onBeat()
}

func (j *Jolteon) ExtraInfo() []byte {
	b1 := common.Uint64ToByte(j.vcp.lastView)
	return append(b1, j.QCHigh.Justify.ToByte()...)
}

func (j *Jolteon) CheckPackAuth(num uint64) bool {
	return j.vcp.lastView%uint64(j.cfg.SaintLen()) == uint64(j.cfg.Turn())
}

func (j *Jolteon) HandleMsg(addr common.Address, code uint, data []byte) {
	switch code {
	case SendVoteCode:
		v := &block.Vote{}
		err := rlp.DecodeBytes(data, v)
		log.Debug("rec SendVoteCode", "vote", v.BC, "user", v.User, "num", v.Number, "status", v.Status)
		if err != nil {
			log.Error("rec SendVoteCode", "err", err)
		}
		if j.Verify(v.Sig, v.RlpHash(), v.User) {
			j.onReceiveVote(v)
		} else {
			log.Error("rec err vote", "v", v)
		}
	case ViewChange:
		vc := &block.ViewChangeQC{}
		err := rlp.DecodeBytes(data, vc)
		log.Debug("rec ViewChange", "vc", vc)
		if err != nil {
			log.Error("rec ViewChange", "err", err)
		}
		//如果收到的高度比自己上一个高度要低，那么可以先把上一个高度传过去
		if vc.LastNum < j.QCHigh.Block.Number-1 {
			j.ResendCurrentBlock(vc.User)
			return
		}
		j.vcp.AddViewChange(vc)
	case ViewChangeAck:
		vca := &block.ViewChangeQCAck{}
		err := rlp.DecodeBytes(data, vca)
		log.Debug("rec ViewChangeAck", "vca", vca)
		if err != nil {
			log.Error("rec ViewChangeAck", "err", err)
		}
		//如果收到的高度比自己上一个高度要低，那么可以先把上一个高度传过去
		if vca.LastNum < j.QCHigh.Block.Number-1 {
			j.ResendCurrentBlock(vca.User)
			return
		}
		j.vcp.AddViewChangeAck(vca)
	case NewView:
		nv := &block.NewViewQC{}
		err := rlp.DecodeBytes(data, nv)
		log.Debug("rec NewView", "nv", nv)
		if err != nil {
			log.Error("rec NewView", "err", err)
		}
		j.onNewView(nv.VcLst)
	}

}

//只需要等待两个区块
func (j *Jolteon) update(bc *block.Block) {
	log.Debug("update", "number", bc.Number, "qcHeight", bc.JolteonQC().Number, "parent", bc.ParentHash, "genesis", j.genesis.Hash)
	preCommitBc := j.getBlock(bc.JolteonQC().BC)
	if preCommitBc == nil {
		log.Error("prepareBc nil!!", "number", bc.Number, "qcHeight", bc.JolteonQC().Number, "parent", bc.ParentHash, "genesis", j.genesis.Hash)
		return
	}
	j.updateQcHigh(&block.QC{
		Block:   preCommitBc,
		Justify: bc.JolteonQC(),
	})
	if preCommitBc.Number > j.BlockLock.Number {
		j.BlockLock = preCommitBc
	}
	commitBc := j.getBlock(preCommitBc.JolteonQC().BC)
	if commitBc == nil {
		return
	}
	if preCommitBc.ParentHash == commitBc.Hash {
		j.onCommit(commitBc)
		j.vcp.addCheckPoint(commitBc, j.QCHigh)
		j.BlockExec = commitBc
	}
}

func (j *Jolteon) updateQcHigh(v *block.QC) {
	log.Debug("updateQcHigh", "block", v.Block.Number, "qcHeight", v.Justify.Number, "local QC", j.QCHigh.Block.Number)
	//在这里，hotstuff允许了未收集满见证的区块继续后面出块，但是只能引用最高的QC
	if v.Block.Number > j.QCHigh.Block.Number {
		j.QCHigh = v
		j.BlockLeaf = v.Block
	}
}

func (j *Jolteon) onCommit(bc *block.Block) {
	if j.BlockExec.Number < bc.Number {
		bc2 := j.getBlock(bc.ParentHash)
		//如果没有父区块，那么就等一等
		if bc2 == nil {
			log.Error("onCommit block parent nil", "bc", bc.Hash, "number", bc.Number)
			return
		}
		j.onCommit(bc2)
		log.Debug("WriteBlock", "bc", bc.Hash, "number", bc.Number)
		//if bc.Number%10000 == 0 {
		//	log.Info("WriteBlock", "bc", bc.Hash, "number", bc.Number)
		//}
		delete(j.blockPool, bc.Hash)
		j.OnBlockConfirm(bc)
	}
}

func (j *Jolteon) onReceiveBlock(bc *block.Block) {
	if bc.Number <= j.vHeight {
		log.Debug("on Receive low Proposal", "receive", bc.Number, "number", j.vHeight)
		return
	}
	//如果是子孙区块，或者是投票和的高度在lock以后，那么就发送投票
	if j.extends(bc, j.BlockLock) || (bc.JolteonQC().Number > j.BlockLock.Number || bc.JolteonQC().Number == 0) {
		j.vHeight = bc.Number
		j.BlockTail = bc
		j.OnBlockAvailable(j.BlockTail)
		leader := j.getLeader()
		log.Debug("SendVote", "leader", leader, "hash", bc.Hash, "number", bc.Number)
		vote := &block.Vote{
			User:   j.cfg.Coinbase(),
			Number: bc.Number,
			Status: 0,
			BC:     bc.Hash,
		}
		sig, _ := j.Signature(vote.RlpHash())
		vote.SetSig(sig)
		if leader == j.cfg.Coinbase() {
			j.update(bc)
			j.onReceiveVote(vote)
		} else {
			j.SendVote(leader, vote)
			j.update(bc)
		}
	} else {
		//更新
		j.update(bc)
	}
	//h.checkVoteEnough(bc.Hash) //不需要吧，因为不会的
}

func (j *Jolteon) getLeader() common.Address {

	return j.cfg.GetSaint(int(j.vcp.lastView % uint64(j.cfg.SaintLen())))
}

func (j *Jolteon) extends(son, father *block.Block) bool {
	for son.Number > father.Number {
		if son.ParentHash == father.Hash {
			return true
		}
		son = j.getBlock(son.ParentHash)
		if son == nil {
			return false
		}
	}
	return son.Hash == father.Hash
}

func (j *Jolteon) onReceiveVote(vote *block.Vote) {
	log.Debug("onReceiveVote", "vote_num", vote.Number, "vote_user", vote.User)
	if j.QCHigh.Justify.Number == vote.Number {
		log.Debug("onReceiveVote", "justify", j.QCHigh.Justify.Number, "vote_num", vote.Number)
		return
	}
	vs, ok := j.votes[vote.BC]
	if !ok {
		vs = make(map[common.Address]*block.Vote)
		j.votes[vote.BC] = vs
	}
	vs[vote.User] = vote
	j.checkVoteEnough(vote.BC)
}

func (j *Jolteon) checkVoteEnough(hash common.Hash) {
	vs, ok := j.votes[hash]
	if !ok {
		return
	}
	bc := j.getBlock(hash)
	if bc == nil {
		log.Debug("onReceiveVote block is nil", "bc", hash)
		return
	}
	//达到阈值
	if len(vs) > j.cfg.Tolerance() {
		log.Debug("receive enough vote", "votes", vs, "len", len(vs), "num", bc.Number,
			"getLeader", j.getLeader(), "self", j.cfg.Coinbase().String())
		voteList := make([]*block.Vote, len(vs))
		i := 0
		for _, v := range vs {
			voteList[i] = v
			i++
		}
		vote := VoteLst2QC(voteList)
		qc := &block.QC{
			Block:   bc,
			Justify: vote,
		}
		j.updateQcHigh(qc)
		if qc.Justify.Number == j.BlockTail.Number {
			j.onBeat()
		}
	}
}

func (j *Jolteon) onBeat() {
	//是当前的节点，就发送提案，然后计时
	leader := j.getLeader()
	log.Debug("onBeat", "leader", leader, "number", j.BlockTail.Number)
	if leader == j.cfg.Coinbase() {
		//log.Info("onBeat", "leader", leader, "number", h.BlockTail.Number)
		j.DoPackBlock()

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
	j.vcp.resetTimer(1)
}

func (j *Jolteon) SendVote(name common.Address, v *block.Vote) {
	j.sender.AdditionSend(name, SendVoteCode, v)
}

func (j *Jolteon) onNewView(qcLst []*block.ViewChangeQC) {
	//取最大的proof
	qc := qcLst[0].LastProof
	for _, vcq := range qcLst {
		if qc.Block.Number <= vcq.LastProof.Block.Number {
			qc = vcq.LastProof
		}
	}

	j.vcp.OnNewView(qc.Block.JolteonView())
	j.updateQcHigh(qc)
	j.onBeat()
}

func VoteLst2QC(vs []*block.Vote) *block.Vote {
	vote := &block.Vote{
		User:   common.Address{},
		Number: vs[0].Number,
		Status: 0,
		BC:     vs[0].BC,
	}
	b, err := rlp.EncodeToBytes(vs)
	if err != nil {
		log.Error("encode voteList fail", "err", err)
	}
	vote.SetSig(b)
	return vote
}

func QC2VoteLst(v *block.Vote) (bool, []*block.Vote) {
	var vs = make([]*block.Vote, 0)
	err := rlp.DecodeBytes(v.Sig, &vs)
	if err != nil {
		log.Error("verifyQC", "err", err)
		return false, nil
	}
	return true, vs
}
