package jolteon

import (
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/consensus/consensus_service"
	"witCon/log"
)

type ViewChangeProcess struct {
	//这些是需要初始化的
	cfg       consensus_service.SaintCluster
	newViewFn func([]*block.ViewChangeQC)
	pm        consensus_service.AdditionSender

	//这是上一次的区块信息
	lastNum   uint64
	lastView  uint64
	lastProof *block.QC

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
	SendVoteCode = iota
	ViewChange
	ViewChangeAck
	NewView
)

func (vcp *ViewChangeProcess) String() string {
	return fmt.Sprintf("viewChange:%v, next is %v, last Block info[n:%v,v:%v]\nvcS:%v.\nvc:%v.\npending:%v",
		vcp.isViewChange, vcp.nextView, vcp.lastNum, vcp.lastView, vcp.vcS, vcp.vc, vcp.pendingVc)
}

func NewViewChangeProcess(cfg consensus_service.SaintCluster, pm consensus_service.AdditionSender,
	newViewFn func([]*block.ViewChangeQC)) *ViewChangeProcess {
	vcp := &ViewChangeProcess{
		cfg:             cfg,
		pm:              pm,
		newViewFn:       newViewFn,
		vcS:             mapset.NewSet(),
		vc:              make(map[common.Address]*ViewChangeState),
		pendingVc:       make(map[uint64]map[common.Address]*ViewChangeState),
		viewChangeCount: 1,
	}
	log.NewGoroutine(vcp.timerLoop)
	return vcp
}

//定时器
func (vpc *ViewChangeProcess) timerLoop() {
	vpc.timer = time.NewTimer(0 * time.Second)
	<-vpc.timer.C
	for {
		select {
		case <-vpc.timer.C:
			log.Error("time loop timeout")
			vpc.occurTimeOut()
		}
	}
}

func (vpc *ViewChangeProcess) resetTimer(count int) {
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

//发生了超时，如果在viewChange过程中，那么就直接发布一个viewChange
//如果不在viewChange的过程中，那么就生成一个新的viewChange，并转换状态
func (vpc *ViewChangeProcess) occurTimeOut() {
	if vpc.isViewChange {
		vpc.nextView++
	} else {
		vpc.nextView = vpc.lastView + 1
	}
	vpc.isViewChange = true
	log.Error("occurTimeOut!")
	vpc.sendViewChange()
	vpc.refreshViewChange()
}

func (vpc *ViewChangeProcess) sendViewChange() {
	//自己的回合是不发viewChange的
	if vpc.isNextViewPrimary() {
		return
	}
	vc := &block.ViewChangeQC{
		LastNum:   vpc.lastNum,
		LastProof: vpc.lastProof,
		View:      vpc.nextView,
		User:      vpc.cfg.Coinbase(),
	}
	vpc.BroadcastViewChange(vc)
	vpc.resetTimer(vpc.viewChangeCount)
	vpc.viewChangeCount *= 2
}

//我们以一个新的区块为一个checkPoint，如果收到一个比当前高度高的新区块，那么就可以停止viewChange
func (vpc *ViewChangeProcess) addCheckPoint(bc *block.Block, commitVotes *block.QC) {
	if bc.Number <= vpc.lastNum {
		return
	}
	vpc.viewChangeCount = 1
	vpc.resetTimer(vpc.viewChangeCount)
	log.Debug("addCheckPoint", "number", bc.Number, "hash", bc.Hash, "view", bc.JolteonView())
	vpc.lastNum = bc.Number
	vpc.lastProof = commitVotes
	vpc.lastView = bc.JolteonView()
	if vpc.isViewChange {
		//取消视图切换
		vpc.isViewChange = false
		vpc.nextView = bc.JolteonView()
		//清理所有小于当前高度的viewChange
		for k, v := range vpc.vc {
			if v.vc != nil && v.vc.LastNum < bc.Number || (v.vc != nil && v.vc.View <= bc.JolteonView()) {
				delete(vpc.vc, k)
			}
		}
		for k, v := range vpc.pendingVc {
			for kk, vv := range v {
				if vv.vc != nil && vv.vc.LastNum < bc.Number || (vv.vc != nil && vv.vc.View <= bc.JolteonView()) {
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

//判断自己是不是当前ViewChange的主节点
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

func (vpc *ViewChangeProcess) AddViewChange(vc *block.ViewChangeQC) {
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
	} else {
		vcs := vpc.getPendingVewChangeState(vc.View, vc.User)
		vcs.setVc(vc)
		//判断是否够多，可以进行viewChange
		vpc.ensurePendingViewChange(vc.View)
	}
}

func (vpc *ViewChangeProcess) AddViewChangeAck(vca *block.ViewChangeQCAck) {
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

//将一个合格的viewChange放入，并判断是否达到发送newView的条件
func (vpc *ViewChangeProcess) ensureViewChange(vc *block.ViewChangeQC) bool {
	vpc.vcS.Add(vc)
	if vpc.vcS.Cardinality() >= vpc.cfg.Tolerance() {
		vpc.sendNewView()
		return true
	}
	return false
}

//如果pending中的viewChange超过了f，就表明需要进行viewChange了
func (vpc *ViewChangeProcess) ensurePendingViewChange(view uint64) {
	vcsMap, ok := vpc.pendingVc[view]
	if ok && len(vcsMap) >= (vpc.cfg.Tolerance()/2) {
		vpc.isViewChange = true
		vpc.nextView = view
		log.Error("ensurePendingViewChange")
		vpc.sendViewChange()
	}
	//然后更新那些viewChange的信息
	vpc.refreshViewChange()
}

//刷新当前统计的vcS等信息
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

func (vpc *ViewChangeProcess) sendViewChangeAck(vc *block.ViewChangeQC) {
	name := vpc.cfg.GetSaint(int(vpc.nextView) % vpc.cfg.SaintLen())
	vpc.SendViewChangeAck(name, &block.ViewChangeQCAck{
		LastNum: vc.LastNum,
		View:    vc.View,
		AckUser: vc.User,
		User:    vpc.cfg.Coinbase(),
	})
}

//发出newView的消息
func (vpc *ViewChangeProcess) sendNewView() {
	vcLst := make([]*block.ViewChangeQC, vpc.vcS.Cardinality())
	for index, vc := range vpc.vcS.ToSlice() {
		vcLst[index] = vc.(*block.ViewChangeQC)
	}
	nv := &block.NewViewQC{
		LastNum: vpc.lastNum,
		NewView: vpc.nextView,
		VcLst:   vcLst,
	}
	vpc.BroadcastNewView(nv)
	vpc.newViewFn(nv.VcLst)
}

func (ps *ViewChangeProcess) BroadcastViewChange(vc *block.ViewChangeQC) {
	ps.pm.AdditionBroadcast(ViewChange, vc)
}

func (ps *ViewChangeProcess) SendViewChangeAck(name common.Address, vca *block.ViewChangeQCAck) {
	ps.pm.AdditionSend(name, ViewChangeAck, vca)
}

func (ps *ViewChangeProcess) BroadcastNewView(nv *block.NewViewQC) {
	ps.pm.AdditionBroadcast(NewView, nv)
}

//用于收集viewChange的Ack
type ViewChangeState struct {
	vc    *block.ViewChangeQC
	f21   int
	vcAck map[common.Address]*block.ViewChangeQCAck
}

func NewViewChangeState(f21 int) *ViewChangeState {
	return &ViewChangeState{
		vc:    nil,
		f21:   f21,
		vcAck: make(map[common.Address]*block.ViewChangeQCAck),
	}
}

func (vcs *ViewChangeState) addAck(vca *block.ViewChangeQCAck) {
	vcs.vcAck[vca.User] = vca
}

func (vcs *ViewChangeState) setVc(vc *block.ViewChangeQC) {
	vcs.vc = vc
}

func (vcs *ViewChangeState) checkValid() bool {
	//因为自己的，和发出viewChange的人都不会ack自己，所以是2f-1
	return vcs.vc != nil && len(vcs.vcAck) >= vcs.f21-1
}
