package verify

import (
	"witCon/common"
	"witCon/common/block"
	"witCon/consensus/consensus_service"
	"witCon/log"
)

type Verify struct {
	consensus_service.ConsensusProxy
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
func NewVerify(proxy consensus_service.ConsensusProxy) *Verify {
	v := &Verify{
		ConsensusProxy: proxy,
	}
	return v
}

func (p *Verify) StartDaemon(daemon common.Address) {
	log.Info("action")
}

func (p *Verify) CommitBlock(bc *block.Block, vs []*block.Vote) {
	log.Debug("confirmBlock", "bc", bc.Hash, "num", bc.Number)
	if bc.Number%1000 == 0 {
		log.Error("confirmBlock", "bc", bc.Hash, "num", bc.Number)
	}
	p.OnBlockConfirm(bc)
	p.OnBlockAvailable(bc)
	log.Debug("OnBlockAvailable")
}

func (p *Verify) CommitSignature(v *block.Vote) {
	return
}

func (p *Verify) ExistBlock(hash common.Hash) bool {
	return false
}

func (p *Verify) GetProcessBlock(hash common.Hash) (interface{}, bool) {
	return nil, false
}

func (p *Verify) HandleMsg(addr common.Address, code uint, data []byte) {
	return
}

func (p *Verify) CheckPackAuth(num uint64) bool {
	return false
}

func (p *Verify) ExtraInfo() []byte {
	return []byte{}
}
