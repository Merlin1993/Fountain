package pbft

import (
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/consensus/consensus_service"
	"witCon/log"
)

const (
	prepare = iota
	commit
)

type PBFT struct {
	consensus_service.ConsensusProxy
	consensus_service.Seal
	nc           consensus_service.SaintCluster
	currentNum   uint64       //当前高度
	currentBlock *block.Block //当前区块
	currentVC    *VoteCollect

	view uint64 //当前视图号

	vcp *ViewChangeProcess
}

//二阶段
//未发出的阶段为 pre-prepare
//发出投票 为 prepare
//收到第一轮2/3 为 commit
//收到第二轮2/3 写入区块，进入下一轮

// 视图切换
// 首先自己发送自己的投票发给大家
// 然后大家回复确认，确认达到2/3完成切换
// 如果大家回复确认以没收到2/3,就更高的高度进行视图切换
func NewPBFT(nc consensus_service.SaintCluster, sender consensus_service.AdditionSender, seal consensus_service.Seal, proxy consensus_service.ConsensusProxy) *PBFT {
	pbft := &PBFT{
		currentNum:     1,
		nc:             nc,
		Seal:           seal,
		ConsensusProxy: proxy,
	}
	pbft.vcp = NewViewChangeProcess(nc, sender, pbft.newView)
	return pbft
}

func (p *PBFT) StartDaemon(daemon common.Address) {
	log.Info("action")
}

func (p *PBFT) CommitBlock(bc *block.Block, vs []*block.Vote) {
	p.receiveBlock(bc, vs)
}

func (p *PBFT) CommitSignature(v *block.Vote) {
	if p.vcp.isViewChange {
		return
	}
	p.receiveVote(v)
}

func (p *PBFT) ExistBlock(hash common.Hash) bool {
	if p.currentBlock != nil && p.currentBlock.Hash == hash {
		return true
	}
	return false
}

func (p *PBFT) GetProcessBlock(hash common.Hash) (interface{}, bool) {
	if p.currentBlock != nil && p.currentBlock.Hash == hash {
		return p.currentBlock, true
	}
	return nil, false
}

func (p *PBFT) newView(newView uint64) {
	p.view = newView
	log.Info("NewView", "view", p.view)
	p.vcp.resetTimer(1)
	p.currentBlock = nil
	p.vcp.OnNewView(newView)
	p.DoPackBlock()
}

func (p *PBFT) HandleMsg(addr common.Address, code uint, data []byte) {
	switch code {
	case ViewChange:
		vc := &block.ViewChange{}
		err := rlp.DecodeBytes(data, vc)
		log.Debug("rec ViewChange", "vc", vc, "from", addr)
		if err != nil {
			log.Error("rec ViewChange", "err", err)
		}
		//如果收到的高度比自己上一个高度要低，那么可以先把上一个高度传过去
		if vc.LastNum < p.currentNum-1 {
			p.ResendCurrentBlock(vc.User)
			return
		}
		p.vcp.AddViewChange(vc)
	case ViewChangeAck:
		vca := &block.ViewChangeAck{}
		err := rlp.DecodeBytes(data, vca)
		log.Debug("rec ViewChangeAck", "vca", vca)
		if err != nil {
			log.Error("rec ViewChangeAck", "err", err)
		}
		//如果收到的高度比自己上一个高度要低，那么可以先把上一个高度传过去
		if vca.LastNum < p.currentNum-1 {
			p.ResendCurrentBlock(vca.User)
			return
		}
		p.vcp.AddViewChangeAck(vca)
	case NewView:
		nv := &block.NewView{}
		err := rlp.DecodeBytes(data, nv)
		log.Debug("rec NewView", "nv", nv)
		if err != nil {
			log.Error("rec NewView", "err", err)
		}
		p.newView(nv.NewView)
	}
}

// -----------------------------下面是新的代码--------------
// 是否是主节点
func (p *PBFT) CheckPackAuth(num uint64) bool {
	/*if p.currentNum != num {
		return false
	}*/
	return p.view%uint64(p.nc.SaintLen()) == uint64(p.nc.Turn())
}

func (p *PBFT) ExtraInfo() []byte {
	return common.Uint64ToByte(p.view)
}

func (p *PBFT) clearCurrent() {
	p.currentNum++
	p.currentBlock = nil
	p.currentVC = nil
}

func (p *PBFT) receiveBlock(bc *block.Block, vs []*block.Vote) {
	//直接写入区块
	if bc.Proof != nil && len(bc.Proof) > 0 {
		//如果是新的view,说明已经完成了view的切换，直接顺着改过来
		p.view = bc.View()
		log.Debug("get proof bc", "num", bc.Number, "hash", bc.Hash)
		p.confirmBlock(bc)
		return
	}
	//如果视图切换中，拒绝接受未确认的区块
	if p.vcp.isViewChange {
		log.Debug("receive block but in viewChange")
		return
	}
	//如果到达当前高度，并且当前view和num相同，说明是正确的区块
	if p.currentBlock == nil && p.currentNum == bc.Number && p.view == bc.View() {
		p.currentBlock = bc
		p.currentVC = newVoteCollectForVotes(p.nc.Tolerance(), vs)

		log.Debug("receive block and vote", "len", len(vs), "prepare", p.currentVC.prepare.Cardinality(), "commit", p.currentVC.commit.Cardinality())
		//是否要记录所有的投票记录进行重发？
		p.sendVote(bc)
	} else {
		log.Debug("can't handle block ", "has currentBlock", p.currentBlock == nil, "view", p.view, "current", p.currentNum)
	}
}

func (p *PBFT) confirmBlock(bc *block.Block) {
	log.Debug("complete confirmBlock", "bc", bc.Hash, "num", bc.Number)
	p.OnBlockConfirm(bc)
	if bc.Number == p.currentNum {
		var proof []*block.Vote
		if p.currentVC != nil {
			proof = make([]*block.Vote, p.currentVC.commit.Cardinality())
			for index, vote := range p.currentVC.commit.ToSlice() {
				proof[index] = vote.(*block.Vote)
			}
		}
		p.clearCurrent()
		p.vcp.addCheckPoint(bc, proof)
	} else {
		p.vcp.addCheckPoint(bc, []*block.Vote{})
	}
	p.OnBlockAvailable(bc)
	log.Debug("OnBlockAvailable")
	p.DoPackBlock()
}

func (p *PBFT) receiveVote(v *block.Vote) {
	//如果是自己就增加投票
	if p.currentBlock != nil && p.currentBlock.Hash == v.BC {
		p.currentVC.addVote(v)
		//然后判断是否达到临界条件
		p.checkStatus()
	}
}

func (p *PBFT) sendVote(bc *block.Block) {
	v := &block.Vote{
		User:   p.nc.Coinbase(),
		Number: bc.Number,
		Status: p.currentVC.status,
		BC:     bc.Hash,
	}
	signature, err := p.Signature(v.RlpHash())
	if err != nil {
		return
	}
	v.SetSig(signature)
	log.Debug("send vote", "bc", bc.Hash, "num", bc.Number, "status", p.currentVC.status)
	p.currentVC.addVote(v)
	p.OnVote(v)
	p.checkStatus()
}

func (p *PBFT) checkStatus() {
	if p.currentVC.isCommitted() {
		p.confirmBlock(p.currentBlock)
		return
	}
	if p.currentVC.isPrepared() && p.currentVC.status == prepare {
		p.currentVC.status = commit
		p.vcp.resetTimer(1)
		p.sendVote(p.currentBlock)
	}
}

// ----------------------针对PBFT设置的结构
type VoteCollect struct {
	prepare mapset.Set
	commit  mapset.Set
	f21     int
	status  uint
}

func newVoteCollect(f21 int) *VoteCollect {
	return &VoteCollect{
		prepare: mapset.NewSet(),
		commit:  mapset.NewSet(),
		f21:     f21,
		status:  prepare,
	}
}

func newVoteCollectForVotes(f21 int, votes []*block.Vote) *VoteCollect {
	vc := &VoteCollect{
		prepare: mapset.NewSet(),
		commit:  mapset.NewSet(),
		f21:     f21,
		status:  prepare,
	}
	for _, v := range votes {
		vc.addVote(v)
	}
	return vc
}

func (vc *VoteCollect) addVote(vote *block.Vote) {
	switch vote.Status {
	case prepare:
		vc.prepare.Add(vote)
	case commit:
		vc.commit.Add(vote)
	}
}

func (vc *VoteCollect) isPrepared() bool {
	return vc.prepare.Cardinality() > vc.f21
}

func (vc *VoteCollect) isCommitted() bool {
	return vc.prepare.Cardinality() > vc.f21 && vc.commit.Cardinality() > vc.f21
}

//--------------视图切换----
//一个计时器，每当写入区块时，重新计时
//当超时时，拒绝进行上述流程（除非其他人写入了区块），进入viewChange，发送viewChange请求
//其他节点收到viewChange，检查其并发送ack给新主节点
//当主节点收到每个view-change的2f+1的viewChange-ack包以后，将view-change加入到区块中。
//当主节点确认2f个view-change后，发送newView消息
//发送新的区块

//对于视图主节点来说，它需要收集自己这个视图下的所有viewchange
//对于非视图主节点来说，它需要判断这个视图切换到底是否需要承认

//对于高度这个变量
//backup 只有对高度大于等于自己的viewChange,才可以回复ack，否则发送checkPoint
//primary 只有对高度大于等于自己的，才可以统计
//如果收到高度更高的区块也就是checkPoint，说明viewChange不生效，重新统计延时

//这里需要注意的是，的确可能有人commit了，但是没人发现，导致那个人可能要回滚，但是因为交易会重放，所以不会出现交易回滚的情况

//只有当前view的主节点才需要收集这个，其他节点只要保留状态就好

type ViewChangeProcess struct {
	//这些是需要初始化的
	cfg       consensus_service.SaintCluster
	newViewFn func(newView uint64)
	pm        consensus_service.AdditionSender

	//这是上一次的区块信息
	lastNum   uint64
	lastView  uint64
	lastProof []*block.Vote

	timer *time.Timer
	//这是和view相关的状态
	isViewChange    bool
	vcS             mapset.Set                                     //已经确定的viewchange消息
	vc              map[common.Address]*ViewChangeState            //处理中的viewchange消息
	pendingVc       map[uint64]map[common.Address]*ViewChangeState //当收到f个新视图的viewChange，就有必要重新切换视图
	nextView        uint64                                         //当前视图切换前往的视图
	viewChangeCount int
}

const (
	ViewChange = iota
	ViewChangeAck
	NewView
)

func (vcp *ViewChangeProcess) String() string {
	return fmt.Sprintf("viewChange:%v, next is %v, last Block info[n:%v,v:%v]\nvcS:%v.\nvc:%v.\npending:%v",
		vcp.isViewChange, vcp.nextView, vcp.lastNum, vcp.lastView, vcp.vcS, vcp.vc, vcp.pendingVc)
}

func NewViewChangeProcess(cfg consensus_service.SaintCluster, pm consensus_service.AdditionSender,
	newViewFn func(newView uint64)) *ViewChangeProcess {
	vcp := &ViewChangeProcess{
		cfg:             cfg,
		pm:              pm,
		newViewFn:       newViewFn,
		vcS:             mapset.NewSet(),
		vc:              make(map[common.Address]*ViewChangeState),
		pendingVc:       make(map[uint64]map[common.Address]*ViewChangeState),
		viewChangeCount: 1,
	}
	vcp.timer = time.NewTimer(0 * time.Second)
	go vcp.timerLoop()
	return vcp
}

// 定时器
func (vpc *ViewChangeProcess) timerLoop() {
	<-vpc.timer.C
	for {
		select {
		case <-vpc.timer.C:

			log.Debug("timerLoop occurTimeOut")
			vpc.occurTimeOut()
		}
	}
}

func (vpc *ViewChangeProcess) resetTimer(count int) {
	log.Debug("reset time", "time", (time.Duration(count) * consensus_service.ViewChangeDuration))
	vpc.timer.Reset(time.Duration(count) * consensus_service.ViewChangeDuration)
}

func (vpc *ViewChangeProcess) OnNewView(nv uint64) {
	if vpc.nextView <= nv {
		vpc.isViewChange = false
		vpc.lastView = nv
		vpc.vcS.Clear()
		vpc.vc = make(map[common.Address]*ViewChangeState)
		delete(vpc.pendingVc, nv)
	}
}

// 发生了超时，如果在viewChange过程中，那么就直接发布一个viewChange
// 如果不在viewChange的过程中，那么就生成一个新的viewChange，并转换状态
func (vpc *ViewChangeProcess) occurTimeOut() {
	if vpc.isViewChange {
		vpc.nextView++
	} else {
		vpc.nextView = vpc.lastView + 1
	}
	vpc.isViewChange = true
	log.Debug("occurTimeOut")
	vpc.sendViewChange()
	vpc.refreshViewChange()
}

func (vpc *ViewChangeProcess) sendViewChange() {
	//自己的回合是不发viewChange的
	if vpc.isNextViewPrimary() {
		return
	}
	vc := &block.ViewChange{
		LastNum:   vpc.lastNum,
		LastProof: vpc.lastProof,
		View:      vpc.nextView,
		User:      vpc.cfg.Coinbase(),
	}
	vpc.BroadcastViewChange(vc)
	vpc.resetTimer(vpc.viewChangeCount)
	vpc.viewChangeCount *= 2
}

// 我们以一个新的区块为一个checkPoint，如果收到一个比当前高度高的新区块，那么就可以停止viewChange
func (vpc *ViewChangeProcess) addCheckPoint(bc *block.Block, commitVotes []*block.Vote) {
	if bc.Number <= vpc.lastNum {
		return
	}
	vpc.viewChangeCount = 1
	vpc.resetTimer(vpc.viewChangeCount)
	log.Debug("addCheckPoint", "number", bc.Number, "hash", bc.Hash, "view", bc.View)
	vpc.lastNum = bc.Number
	vpc.lastProof = commitVotes
	vpc.lastView = bc.View()
	if vpc.isViewChange {
		//取消视图切换
		vpc.isViewChange = false
		vpc.nextView = bc.View()
		//清理所有小于当前高度的viewChange
		for k, v := range vpc.vc {
			if v.vc.LastNum < bc.Number || v.vc.View <= bc.View() {
				delete(vpc.vc, k)
			}
		}
		for k, v := range vpc.pendingVc {
			for kk, vv := range v {
				if vv.vc.LastNum < bc.Number || vv.vc.View <= bc.View() {
					delete(v, kk)
				}
			}
			if len(v) == 0 {
				delete(vpc.pendingVc, k)
			}
		}
		//然后检查剩余的viewChange,如果还有，那么还是要发出viewChange的
		if len(vpc.vc) != 0 {
			vpc.occurTimeOut()
		}
	}
}

// 判断自己是不是当前ViewChange的主节点
func (vpc *ViewChangeProcess) isNextViewPrimary() bool {
	return vpc.isViewPrimary(int(vpc.nextView))
}

func (vpc *ViewChangeProcess) isViewPrimary(view int) bool {
	return view%vpc.cfg.SaintLen() == int(vpc.cfg.Turn())
}

func (vpc *ViewChangeProcess) getViewChangeState(user common.Address) *ViewChangeState {
	vcs, ok := vpc.vc[user]
	if !ok {
		vcs = NewViewChangeState(vpc.cfg.Tolerance())
		vpc.vc[user] = vcs
	}
	return vcs
}

func (vpc *ViewChangeProcess) getPendingVewChangeState(view uint64, user common.Address) *ViewChangeState {
	vcsMap, ok := vpc.pendingVc[view]
	if !ok {
		vcsMap = make(map[common.Address]*ViewChangeState)
		vpc.pendingVc[view] = vcsMap
	}
	vcs, ok := vcsMap[user]
	if !ok {
		vcs = NewViewChangeState(vpc.cfg.Tolerance())
		vcsMap[user] = vcs
	}
	return vcs
}

func (vpc *ViewChangeProcess) AddViewChange(vc *block.ViewChange) {
	log.Debug("AddViewChange", "vc", vc, "vpc", vpc.String())
	if vc.View == vpc.nextView {
		vcs := vpc.getViewChangeState(vc.User)
		vcs.setVc(vc)
		vpc.sendViewChangeAck(vc)
		//如果合格，就放入集合
		if vcs.checkValid() {
			vpc.ensureViewChange(vcs.vc)
			delete(vpc.vc, vcs.vc.User)
		}
	} else if vc.View > vpc.nextView {
		vcs := vpc.getPendingVewChangeState(vc.View, vc.User)
		vcs.setVc(vc)
		//判断是否够多，可以进行viewChange
		vpc.ensurePendingViewChange(vc.View)
	}
}

func (vpc *ViewChangeProcess) AddViewChangeAck(vca *block.ViewChangeAck) {
	log.Debug("AddViewChangeAck", "vca", vca, "vpc", vpc.String())
	if vca.View == vpc.nextView {
		vcs := vpc.getViewChangeState(vca.AckUser)
		vcs.addAck(vca)
		if vcs.checkValid() {
			vpc.ensureViewChange(vcs.vc)
			delete(vpc.vc, vcs.vc.User)
		}
	} else {
		//如果不是主节点，原则上都不用收集的
		vcs := vpc.getPendingVewChangeState(vca.View, vca.AckUser)
		vcs.addAck(vca)
	}
}

// 将一个合格的viewChange放入，并判断是否达到发送newView的条件
func (vpc *ViewChangeProcess) ensureViewChange(vc *block.ViewChange) bool {
	vpc.vcS.Add(vc)
	if vpc.vcS.Cardinality() >= vpc.cfg.Tolerance() {
		vpc.sendNewView()
		return true
	}
	return false
}

// 如果pending中的viewChange超过了f，就表明需要进行viewChange了
func (vpc *ViewChangeProcess) ensurePendingViewChange(view uint64) {
	vcsMap, ok := vpc.pendingVc[view]
	if ok && len(vcsMap) >= (vpc.cfg.Tolerance()/2) {
		vpc.isViewChange = true
		vpc.nextView = view
		log.Debug("ensurePendingViewChange")
		vpc.sendViewChange()
	}
	//然后更新那些viewChange的信息
	vpc.refreshViewChange()
}

// 刷新当前统计的vcS等信息
func (vpc *ViewChangeProcess) refreshViewChange() {
	vpc.vcS.Clear()
	pendingVc, ok := vpc.pendingVc[vpc.nextView]
	if !ok {
		return
	}
	for _, v := range pendingVc {
		//如果自己是这个视图的主节点，就对其进行统计
		if vpc.isNextViewPrimary() {
			if v.checkValid() {
				sended := vpc.ensureViewChange(v.vc)
				if sended {
					break
				}
			} else {
				//不合格，就放到待完成的集合中
				//vpc.vc[v.vc.User] = v
			}
		} else {
			//否则就发送ack
			if v.vc != nil {
				vpc.sendViewChangeAck(v.vc)
			}
		}
	}
	delete(vpc.pendingVc, vpc.nextView)
}

func (vpc *ViewChangeProcess) sendViewChangeAck(vc *block.ViewChange) {
	name := vpc.cfg.GetSaint(int(vpc.nextView) % vpc.cfg.SaintLen())
	vpc.SendViewChangeAck(name, &block.ViewChangeAck{
		LastNum: vc.LastNum,
		View:    vc.View,
		AckUser: vc.User,
		User:    vpc.cfg.Coinbase(),
	})
}

// 发出newView的消息
func (vpc *ViewChangeProcess) sendNewView() {
	vcLst := make([]*block.ViewChange, vpc.vcS.Cardinality())
	for index, vc := range vpc.vcS.ToSlice() {
		vcLst[index] = vc.(*block.ViewChange)
	}
	nv := &block.NewView{
		LastNum: vpc.lastNum,
		NewView: vpc.nextView,
		VcLst:   vcLst,
	}
	vpc.BroadcastNewView(nv)
	vpc.newViewFn(nv.NewView)
}

func (ps *ViewChangeProcess) BroadcastViewChange(vc *block.ViewChange) {
	ps.pm.AdditionBroadcast(ViewChange, vc)
}

func (ps *ViewChangeProcess) SendViewChangeAck(name common.Address, vca *block.ViewChangeAck) {
	ps.pm.AdditionSend(name, ViewChangeAck, vca)
}

func (ps *ViewChangeProcess) BroadcastNewView(nv *block.NewView) {
	ps.pm.AdditionBroadcast(NewView, nv)
}

// 用于收集viewChange的Ack
type ViewChangeState struct {
	vc    *block.ViewChange
	f21   int
	vcAck map[common.Address]*block.ViewChangeAck
}

func NewViewChangeState(f21 int) *ViewChangeState {
	return &ViewChangeState{
		vc:    nil,
		f21:   f21,
		vcAck: make(map[common.Address]*block.ViewChangeAck),
	}
}

func (vcs *ViewChangeState) addAck(vca *block.ViewChangeAck) {
	vcs.vcAck[vca.User] = vca
}

func (vcs *ViewChangeState) setVc(vc *block.ViewChange) {
	vcs.vc = vc
}

func (vcs *ViewChangeState) checkValid() bool {
	//因为自己的，和发出viewChange的人都不会ack自己，所以是2f-1
	return vcs.vc != nil && len(vcs.vcAck) >= vcs.f21-1
}
