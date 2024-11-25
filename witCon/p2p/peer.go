package p2p

import (
	lru "github.com/hashicorp/golang-lru"
	"sync"
	"time"
	"witCon/common"
)

type Peer struct {
	conn        *Conn
	node        *Node
	name        common.Address
	receiveVote *lru.ARCCache
	shard       []uint
}

const (
	BaseSuite = iota
	ConsensusSuite
)

const (
	HandshakeCode = iota
	RequestName
	Action
	De1
	De2
	De3
	De4
	De5
	de6
	ConnectEOF
)

func mixProtocol(suite uint, code uint) uint {
	return suite<<8 + code
}

func resolveProtocol(proto uint) (suite uint, code uint) {
	return proto >> 8, proto % (1 << 8)
}

type Handshake struct {
	Addr   common.Address
	Shards []uint
}

func (p *Peer) Start(name common.Address, handle func(code uint, data []byte)) {
	p.receiveVote, _ = lru.NewARC(1000)
	p.conn.ReadQuicMsgLoop(handle)
	p.SendHandshake(name, common.VerifyNode)
	p.tryGetName(name)
}

// 重发获取名字
func (p *Peer) tryGetName(name common.Address) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		if p.name == common.EmptyAddress {
			p.conn.SendQuicMsg(mixProtocol(BaseSuite, RequestName), &struct{}{})
			p.tryGetName(name)
		}
	}()
}

func (p *Peer) Send(code uint, data interface{}) {
	p.conn.SendQuicMsg(code, data)
}

func (p *Peer) SendHandshake(name common.Address, shards []uint) {
	p.conn.SendQuicMsg(mixProtocol(BaseSuite, HandshakeCode), &Handshake{Addr: name, Shards: shards})
}

func (p *Peer) SendAction() {
	p.conn.SendQuicMsg(mixProtocol(BaseSuite, Action), struct{}{})
}

type PeerSet struct {
	mapLock             sync.RWMutex
	peerMap             map[common.Address]*Peer
	actionMap           map[common.Address]struct{}
	requestBlockHistory map[common.Hash]int
}

func NewPeerSet() *PeerSet {
	ps := &PeerSet{peerMap: make(map[common.Address]*Peer), actionMap: make(map[common.Address]struct{})}
	ps.requestBlockHistory = make(map[common.Hash]int)
	return ps
}

func (ps *PeerSet) CheckAmount(size int) bool {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	if len(ps.peerMap) == (size - 1) {
		return true
	}
	return false
}

func (ps *PeerSet) AddPeer(name common.Address, peer *Peer) {
	ps.mapLock.Lock()
	defer ps.mapLock.Unlock()
	ps.peerMap[name] = peer
}

func (ps *PeerSet) GetPeer(name common.Address) *Peer {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	return ps.peerMap[name]
}

func (ps *PeerSet) BroadcastVerifyNode(suite uint, code uint, verifyDataFn func([]uint) interface{}) {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	for _, v := range ps.peerMap {
		if v.shard != nil && len(v.shard) > 0 {
			v.Send(mixProtocol(suite, code), verifyDataFn(v.shard))
		}
	}
}

func (ps *PeerSet) BroadcastConsensusNode(suite uint, code uint, data interface{}) {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	for _, v := range ps.peerMap {
		if len(v.shard) == 0 {
			v.Send(mixProtocol(suite, code), data)
		}
	}
}

func (ps *PeerSet) Send(addr common.Address, suite uint, code uint, data interface{}) {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	p, ok := ps.peerMap[addr]
	if ok {
		p.Send(mixProtocol(suite, code), data)
	}
}

func (ps *PeerSet) VerifyNode(addr common.Address) []uint {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	p, ok := ps.peerMap[addr]
	if ok {
		return p.shard
	} else {
		return nil
	}
}

func (ps *PeerSet) BroadcastAction() {
	ps.mapLock.RLock()
	defer ps.mapLock.RUnlock()
	for _, p := range ps.peerMap {
		p.SendAction()
	}
}
