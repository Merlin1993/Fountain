package consensus

import (
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/consensus/consensus_service"
	"witCon/log"
	"witCon/p2p"
)

const (
	SendBlockCode = iota
	SendVoteCode
	ReSendVoteCode
	RequestBlock
	SendCBlockCode
	BaseOffset
)

const ConsensusSuite = p2p.ConsensusSuite

type ProtocolSender interface {
	BroadcastVerifyNode(suite uint, code uint, verifyDataFn func([]uint) interface{})
	BroadcastConsensusNode(suite uint, code uint, data interface{})
	Send(addr common.Address, suite uint, code uint, data interface{})
	VerifyNode(addr common.Address) []uint
}

type ProtoHandler struct {
	ProtocolSender
	proxy           *Proxy
	additionHandler consensus_service.AdditionHandler

	size                int
	requestBlockHistory map[common.Hash]int
}

func newProtoHandle(size int, proxy *Proxy) *ProtoHandler {
	return &ProtoHandler{
		requestBlockHistory: make(map[common.Hash]int),
		size:                size,
		proxy:               proxy,
	}
}

func (ps *ProtoHandler) AdditionSend(addr common.Address, code uint64, data interface{}) {
	ps.Send(addr, ConsensusSuite, uint(BaseOffset+code), data)
}

func (ps *ProtoHandler) AdditionBroadcast(code uint, data interface{}) {
	ps.BroadcastConsensusNode(ConsensusSuite, uint(BaseOffset+code), data)
}

func (ps *ProtoHandler) SetAdditionHandle(additionHandler consensus_service.AdditionHandler) {
	ps.additionHandler = additionHandler
}

func (ps *ProtoHandler) BroadcastCBlock(verifyDataFn func([]uint) interface{}) {
	ps.BroadcastVerifyNode(ConsensusSuite, SendCBlockCode, verifyDataFn)
}

func (ps *ProtoHandler) BroadcastBlock(bc *block.Block) {
	//判断共识节点是否也是分片验证的方式验证所有交易
	cbc := bc.ShallowCopyBC()
	if common.ShardVerify {
		cbc.ShardBody = bc.ShardBody
	}

	ps.BroadcastConsensusNode(ConsensusSuite, SendBlockCode, bc)
}

func (ps *ProtoHandler) BroadcastVote(v *block.Vote) {
	ps.BroadcastConsensusNode(ConsensusSuite, SendVoteCode, v)
}

func (ps *ProtoHandler) BroadcastResendVote(v *block.Vote) {
	ps.BroadcastConsensusNode(ConsensusSuite, ReSendVoteCode, v)
}

func (ps *ProtoHandler) RequestBlock(name common.Address, hash common.Hash) {
	//加上这个，是避免太快的请求区块
	_, ok := ps.requestBlockHistory[hash]
	if !ok {
		ps.requestBlockHistory[hash] = 0
	}
	//主要用于避免所有节点都向自己发送信息，导致请求了很多遍
	if ps.size > 0 && ps.requestBlockHistory[hash]%ps.size == 0 {
		log.Debug("request block", "hash", hash)
		ps.Send(name, ConsensusSuite, RequestBlock, hash)
	}
	ps.requestBlockHistory[hash] += 1
}

func (ps *ProtoHandler) SendBlock(name common.Address, bc *block.Block) {
	shards := ps.VerifyNode(name)
	if len(shards) > 0 {
		sbc := bc.ShallowCopyShard(shards)
		ps.Send(name, ConsensusSuite, SendCBlockCode, sbc)
	} else {
		ps.Send(name, ConsensusSuite, SendBlockCode, bc)
	}

}

func (ps *ProtoHandler) SendVote(name common.Address, v *block.Vote) {
	ps.Send(name, ConsensusSuite, SendVoteCode, v)
}

type requestBlock struct {
	name common.Address
	hash common.Hash
}

func (p *ProtoHandler) Start(ps ProtocolSender) {
	p.ProtocolSender = ps
}

func (p *ProtoHandler) HandleMsg(addr common.Address, code uint, data []byte) {
	if code < BaseOffset {
		switch code {
		case SendBlockCode:
			bc := &block.Block{}
			err := rlp.DecodeBytes(data, bc)
			log.Debug("rec SendBlockCode", "number", bc.Number, "hash", bc.Hash)
			if err != nil {
				log.Error("rec SendBlockCode", "err", err)
			}

			p.proxy.OnSendBlock(&SendBlock{
				name: addr,
				bc:   bc,
			})

		case SendCBlockCode:
			bc := &block.Block{}
			err := rlp.DecodeBytes(data, bc)
			log.Debug("rec SendCBlockCode", "number", bc.Number, "hash", bc.Hash)
			if err != nil {
				log.Error("rec SendCBlockCode", "err", err)
			}

			p.proxy.OnSendBlock(&SendBlock{
				name: addr,
				bc:   bc,
			})
		case SendVoteCode:
			v := &block.Vote{}
			err := rlp.DecodeBytes(data, v)
			log.Debug("rec SendVoteCode", "vote", v.BC, "user", v.User, "num", v.Number, "status", v.Status)
			if err != nil {
				log.Error("rec SendVoteCode", "err", err)
			}
			if p.proxy.Verify(v.Sig, v.RlpHash(), v.User) {
				p.proxy.SynchronizeRun(func() {
					p.proxy.OnSendVote(v)
				})
			} else {
				log.Error("protocol rec err vote", "v", v)
			}
		case ReSendVoteCode:
			v := &block.Vote{}
			err := rlp.DecodeBytes(data, v)
			log.Debug("rec ReSendVoteCode", "vote", v.BC, "user", v.User)
			if err != nil {
				log.Error("rec SendVoteCode", "err", err)
			}
			p.proxy.SynchronizeRun(func() {
				p.proxy.OnReSendVote(v)
			})
		case RequestBlock:
			h := common.Hash{}
			err := rlp.DecodeBytes(data, &h)
			log.Debug("rec RequestBlock", "hash", h)
			if err != nil {
				log.Error("rec RequestBlock", "err", err)
			}
			p.proxy.SynchronizeRun(func() {
				p.proxy.OnRequestBlock(&requestBlock{
					name: addr,
					hash: h,
				})
			})
		}
	} else {
		p.proxy.SynchronizeRun(func() {
			p.additionHandler.HandleMsg(addr, code-BaseOffset, data)
		})
	}
}
