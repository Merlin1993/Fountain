package consensus

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"sync"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/consensus/consensus_service"
	"witCon/consensus/hotstuff"
	"witCon/consensus/jolteon"
	"witCon/consensus/pbft"
	"witCon/consensus/symphony"
	"witCon/consensus/verify"
	"witCon/log"
	"witCon/stat"
)

const (
	Symphony = iota
	pBFT
	hotStuff
	Jolteon
)

type Proxy struct {
	*ProtoHandler
	consensus_service.Seal
	ci     consensus_service.ConsensusImpl
	cfg    consensus_service.SaintCluster
	crw    ChainRW
	syncer Syncer
	pool   consensus_service.TxPool
	state  consensus_service.WorldState

	genesis       *block.Block
	lastHash      common.Hash
	lastBlock     *block.Block
	lastNumber    uint64
	writeNumber   uint64
	taskChan      chan func() //待分配任务的通道
	broadcastChan chan func() //待分配广播的通道
	startBCChan   chan *block.Block

	verifyPool *ants.Pool
	sigPool    *ants.Pool
	prePack    bool
	//packCH       chan *block.Block
}

func NewProxy(nc consensus_service.SaintCluster, crw ChainRW, t uint, seal consensus_service.Seal, pool consensus_service.TxPool, state consensus_service.WorldState) *Proxy {
	p := &Proxy{
		ProtoHandler:  nil,
		Seal:          seal,
		cfg:           nc,
		crw:           crw,
		taskChan:      make(chan func(), 10000),
		broadcastChan: make(chan func(), 10000),
		pool:          pool,
		state:         state,
		prePack:       common.PrePacked,
		startBCChan:   make(chan *block.Block, 1),
	}
	p.genesis = block.NewBlock(0, common.Hash{}, common.Uint64ToByte(0), common.EmptyAddress)
	p.crw.SetGenesis(p.genesis)
	p.lastHash = p.genesis.Hash
	p.syncer = NewSyncerImpl(p, p.crw)
	p.ProtoHandler = newProtoHandle(nc.SaintLen(), p)
	if len(common.VerifyNode) > 0 {
		pm := verify.NewVerify(p)
		p.ci = pm
		p.ProtoHandler.SetAdditionHandle(pm)
	} else {
		switch t {
		case Symphony:
			pm := symphony.NewWit(nc, seal, p, p.genesis)
			p.ci = pm
			p.ProtoHandler.SetAdditionHandle(pm)
		case pBFT:
			c := pbft.NewPBFT(nc, p.ProtoHandler, seal, p)
			p.ci = c
			p.ProtoHandler.SetAdditionHandle(c)
		case hotStuff:
			hs := hotstuff.NewHotstuff(nc, p.ProtoHandler, p.genesis, seal, p)
			p.ci = hs
			p.ProtoHandler.SetAdditionHandle(hs)
		case Jolteon:
			j := jolteon.NewJolteon(nc, p.ProtoHandler, p.genesis, seal, p)
			p.ci = j
			p.ProtoHandler.SetAdditionHandle(j)
		}
	}
	p.verifyPool, _ = ants.NewPool(int(common.ShardVerifyCore))
	p.sigPool, _ = ants.NewPool(int(common.SignVerifyCore))
	log.NewGoroutine(p.mainLoop)
	log.NewGoroutine(p.packLoop)
	log.NewGoroutine(p.broadcastLoop)
	return p
}

func (p *Proxy) mainLoop() {
	for {
		select {
		case task := <-p.taskChan:
			//因为区块上面的差异较大，不好将代码集成到一起，所以这里使用这种方式切换go程
			task()
		}
	}
}

func (p *Proxy) broadcastLoop() {
	for {
		select {
		case task := <-p.broadcastChan:
			//因为区块上面的差异较大，不好将代码集成到一起，所以这里使用这种方式切换go程
			task()
		}
	}
}

func (p *Proxy) Init(ps ProtocolSender) {
	p.ProtoHandler.Start(ps)
	//p.packBlock(p.genesis, true)
}

func (p *Proxy) Action() {
	//只有第一个节点需要读交易
	if p.cfg.GetSaintTurn(p.cfg.Coinbase()) == 0 {
		log.NewGoroutine(func() { p.pool.StartTxRate(common.TxSize) })
	}
	log.Debug("Proxy action before")
	p.SynchronizeRun(func() {
		log.Debug("Proxy action")
		//time.Sleep(8 * time.Second)
		p.ci.StartDaemon(p.cfg.Coinbase())
		p.startBCChan <- p.genesis
	})
}

func (p *Proxy) OnSendBlock(sb *SendBlock) {
	p.syncer.CheckBlockRef(sb.name, sb.bc)
}

func (p *Proxy) OnSendVote(v *block.Vote) {
	if p.syncer.CheckVoteExist(v) {
		p.ci.CommitSignature(v)
	}
}

func (p *Proxy) OnReSendVote(v *block.Vote) {
	p.ci.CommitSignature(v)
}

func (p *Proxy) OnRequestBlock(rb *requestBlock) {
	bc := p.crw.GetBlock(rb.hash)
	log.Debug("handle requestBlock", "hash", rb.hash, "name", rb.name, "bc", bc == nil)
	if bc != nil {
		p.SendBlock(rb.name, bc)
		return
	}
	val, ok := p.ci.GetProcessBlock(rb.hash)
	if ok {
		bc = val.(*block.Block)
		p.SendBlock(rb.name, bc)
		return
	}

	if p.lastBlock != nil && p.lastBlock.Hash == rb.hash {
		p.SendBlock(rb.name, p.lastBlock)
		return
	}
}

//func (p *Proxy) OnBranchExist(ProofBlock) {}

func (p *Proxy) OnBlockConfirm(bc *block.Block) {

	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "OnBlockConfirm")
	p.state.OnBlockConfirm(bc)
	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "OnBlockConfirmStateBlock")
	p.writeBlock(bc)
	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "OnBlockConfirmwriteBlock")
	p.BroadcastRun(func() {
		p.BroadcastCBlock(func(shard []uint) interface{} {
			return bc.ShallowCopyShard(shard)
		})
	})

	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "BroadcastRun")

}

func (p *Proxy) OnBlockCFTConfirm(bc *block.Block) {
	stat.Instance.OnBlockConfirm(bc)
}

// 可用就开始打包下一个区块
func (p *Proxy) OnBlockAvailable(bc *block.Block) {
	p.lastHash = bc.Hash
	p.lastBlock = bc
	p.lastNumber = bc.Number
	p.syncer.OnWriteBlock(bc)
}

func (p *Proxy) OnVote(v *block.Vote) {
	p.ProtoHandler.BroadcastVote(v)
}

//func (p *Proxy) OnBlockProcess(ProcessBlock) {}

func (p *Proxy) SynchronizeRun(task func()) {
	p.taskChan <- task
}

func (p *Proxy) BroadcastRun(task func()) {
	p.broadcastChan <- task
}

func (p *Proxy) HasAvailableBlock(hash common.Hash, num uint64) bool {
	if p.ci.ExistBlock(hash) && p.lastNumber >= num {
		return true
	}
	return p.crw.GetBlock(hash) != nil || num == 0
}

func (p *Proxy) HasBlock(hash common.Hash) bool {
	return p.ci.ExistBlock(hash) || p.crw.GetBlock(hash) != nil
}

func (p *Proxy) CommitBlock(bc *block.Block) {
	p.SynchronizeRun(func() {
		p.commitBlock(bc)
	})
}

func (p *Proxy) VerifyBlock(bc *block.Block) bool {
	t1 := time.Now().UnixMilli()
	if bc.Number <= p.writeNumber {
		log.Error("low bc", "bc", bc.Number, "write", p.writeNumber)
		return false
	}
	//如果没有交易，考虑吧shardBody中的交易给取出来。
	if len(bc.Txs) == 0 {
		bc.Txs = make([]*block.Transaction, 0)
		for _, sb := range bc.ShardBody {
			bc.Txs = append(bc.Txs, sb.Txs...)
		}
	}
	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "start")
	var wg sync.WaitGroup
	var sigErr error

	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "sigErr")
	go func() {
		wg.Add(1)
		if common.SignatureVerify {
			succ, err := VerifyTxsEcdsaSigMul(bc.Txs, p.sigPool)
			if !succ {
				log.Error("verify txs fail", "err", err)
				sigErr = err
			}
			//crypto.VerifySignature()
		} else {
			//实验不签名的效果
			//succ, err := VerifyTxsEcdsaSig(bc.Txs)
			//if !succ {
			//	log.Error("verify txs fail", "err", err)
			//	sigErr = err
			//}
		}
		stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "VerifyTxsEcdsaSigMul")
		wg.Done()
	}()

	//if len(bc.Txs) > 0 {
	//	log.Error("verify txs  len", "len", len(bc.Txs))
	//}
	if common.ShardVerify || len(common.VerifyNode) > 0 {
		err := p.state.VerifyShardMulti(bc, bc.ShardBody, p.verifyPool)
		if err != nil {
			log.Error("verify shard fail", "err", err)
			return false
		}
		stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "VerifyShardMulti")
	} else {
		sbl, root, err := p.state.ExecuteBc(bc, bc.Txs)
		if err != nil {
			log.Error("verify bc fail", "err", err)
			return false
		}
		//sbl, root := p.state.CommitRoot(vm)
		if root != bc.LedgerHash {
			log.Crit("verify bc not equal", "number", bc.Number, "root", root, "ledgerHash", bc.LedgerHash,
				"len", len(bc.Txs), "sbl", fmt.Sprintf("%v", sbl))
		}
		bc.ShardBody = sbl
	}
	wg.Wait()
	if sigErr != nil {
		return false
	}
	t2 := time.Now().UnixMilli()
	if len(bc.Txs) > 0 {
		log.Debug("verify", "bc", bc.Number, "tx", len(bc.Txs), "time", t2-t1)
	}
	return true
}

func (p *Proxy) commitBlock(bc *block.Block) {
	vs := p.syncer.GetTempVote(bc.Hash)
	stat.Instance.OnBlockIn(bc.Hash)
	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "OnBlockIn")
	p.ci.CommitBlock(bc, vs)
	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(bc.Txs), "CommitBlock")
}

func (p *Proxy) writeBlock(bc *block.Block) {
	//存储数据
	//if bc.Number%10000 == 0 {
	//	log.Info("writeBlock", "number", bc.Number, "hash", bc.Hash, "parentHash", bc.ParentHash, "payload", len(bc.Payload))
	//}
	p.writeNumber = bc.Number
	p.crw.WriteBlock(bc)
}

func (p *Proxy) packLoop() {
	var pendingBlock = <-p.startBCChan
	for {
		if pendingBlock.Number-p.writeNumber > 10 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		b1 := p.packBlock(pendingBlock)
		if b1 != nil {
			if len(b1.Txs) == 0 {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			pendingBlock = b1
			nbc := pendingBlock
			p.SynchronizeRun(func() {
				if p.writeNumber+1 == nbc.Number {
					stat.Instance.OnBlockIn(nbc.Hash)
					p.ci.CommitBlock(nbc, nil)
				} else {
					p.syncer.AddPendingBlock(nbc)
				}
				p.ProtoHandler.BroadcastBlock(nbc)
			})
		} else {
			break
		}
	}
}

func (p *Proxy) packBlock(bc *block.Block) *block.Block {
	if p.ci.CheckPackAuth(bc.Number + 1) {
		log.Debug("pack block", "number", bc.Number+1, "parent hash", bc.Hash)
		nbc := block.NewBlock(bc.Number+1, bc.Hash, p.ci.ExtraInfo(), p.cfg.Coinbase())
		txs := p.pool.ReadTx(nbc.ParentHash, nbc.Hash, common.TxAmount)
		if len(txs) > 0 {
			stat.DBlockTimeTrace.AddDBTime(nbc.Number, len(txs), "start")
			log.Warn("read txs", "size", len(txs), "number", bc.Number+1)
		}
		nbc.SetTxs(txs)
		stat.DBlockTimeTrace.AddDBTime(nbc.Number, len(txs), "SetTxs")
		sbl, root, err := p.state.ExecuteBc(nbc, txs)
		stat.DBlockTimeTrace.AddDBTime(nbc.Number, len(txs), "ExecuteBc")
		if err != nil {
			log.Error("pack block fail by execute txs", "err", err)
			return nil
		}
		//sbl, root := p.state.CommitRoot(vm)
		log.Info("pack block", "number", nbc.Number, "tx len", len(nbc.Txs), "root", root)

		log.Debug("pack block execute", "root", root)
		nbc.SetLedgerHash(root)
		nbc.ShardBody = sbl

		return nbc
	} else {
		log.Debug("no auth pack block")
	}
	return nil
}

func (p *Proxy) getLastBlock() *block.Block {
	bc := p.crw.GetBlock(p.lastHash)
	if bc == nil {
		val, ok := p.ci.GetProcessBlock(p.lastHash)
		if ok {
			bc = val.(*block.Block)
		} else {
			return p.lastBlock
		}
	}
	return bc
}

// pbft
func (p *Proxy) ResendCurrentBlock(addr common.Address) {
	bc := p.getLastBlock()
	if bc != nil {
		p.SendBlock(addr, bc)
	}
}

func (p *Proxy) DoPackBlock() {
	bc := p.getLastBlock()
	if bc == nil {
		log.Error("last block is nil")
		return
	}
	log.Debug("send bc", "number", bc.Number, "parent hash", bc.Hash)
	//p.packCH <- bc
	//DO Nothing
}

//func (p *Proxy) PackLoop() {
//	for {
//		select {
//		case bc := <-p.packCH:
//			if p.prePackBlock != nil && p.prePackBlock.Number == bc.Number+1 {
//				log.Debug("pre pack block", "number", bc.Number, "parent hash", bc.Hash)
//
//				stat.DBlockTimeTrace.AddDBTime(p.prePackBlock.Number, len(p.prePackBlock.Txs), "startbroadcastBlock")
//				p.broadcastBlock(p.prePackBlock)
//				stat.DBlockTimeTrace.AddDBTime(p.prePackBlock.Number, len(p.prePackBlock.Txs), "endbroadcastBlock")
//				if !p.cfg.Rotation() && p.prePack {
//					p.packBlock(p.prePackBlock, true)
//				}
//				continue
//			}
//			log.Debug("repack block", "number", bc.Number, "parent hash", bc.Hash)
//			nbc := p.packBlock(bc, false)
//			if nbc == nil {
//				continue
//			}
//			if !p.cfg.Rotation() && p.prePack {
//				p.packBlock(nbc, true)
//			}
//		}
//	}
//
//}
