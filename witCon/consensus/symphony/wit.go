package symphony

import (
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"sync"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/consensus/consensus_service"
	"witCon/log"
)

type Wit struct {
	consensus_service.ConsensusProxy
	consensus_service.Seal
	cfg          consensus_service.SaintCluster
	branches     map[uint64]*branch //分叉
	pendingBlock sync.Map           //map[common.Hash]*block.Block //所有区块
	//pendingNumberBlock map[uint64]map[common.Hash]*block.Block //待确定的高度
	lastEnsureBlock *block.Block //最后一次认定的父区块

	//lastNumber     uint64 //已经确定的高度
	lastCFTBlockHash common.Hash //最后CFT确认的hash
	lastBlock        *block.Block
	//blockCache     *lru.ARCCache
	lastVoteNumber uint64

	usedVote map[uint64]*block.Vote //之前进行过的投票
	//tempNode map[common.Hash]*branch //之前收集过的区块，但是因为没有父区块而无法使用,key为父hash，value是引用的子区块节点集合

	packTimer *time.Timer

	roundOneNumber uint64
	roundTwoNumber uint64
	roundOne       *block.Block
	roundTwo       *block.Block
}

//如果没有收集全的问题
func NewWit(nc consensus_service.SaintCluster, seal consensus_service.Seal, proxy consensus_service.ConsensusProxy, lastBlock *block.Block) *Wit {
	wit := &Wit{
		cfg:      nc,
		Seal:     seal,
		branches: make(map[uint64]*branch),
		//pendingBlock: make(map[common.Hash]*block.Block),
		//pendingNumberBlock: make(map[uint64]map[common.Hash]*block.Block),
		usedVote:         make(map[uint64]*block.Vote),
		ConsensusProxy:   proxy,
		lastBlock:        lastBlock,
		lastCFTBlockHash: common.Hash{},
	}
	//wit.blockCache, _ = lru.NewARC(1000)
	go wit.packLoop()
	return wit
}

func (p *Wit) HandleMsg(addr common.Address, code uint, data []byte) {
	return
}

func (p *Wit) CommitSignature(v *block.Vote) {
	p.receiveVote(v)
}

func (p *Wit) ExistBlock(hash common.Hash) bool {
	_, ok := p.pendingBlock.Load(hash)
	return ok
}

func (p *Wit) GetProcessBlock(hash common.Hash) (interface{}, bool) {
	bc, ok := p.pendingBlock.Load(hash)
	return bc.(*block.Block), ok
}

func (p *Wit) CommitBlock(bc *block.Block, vs []*block.Vote) {
	log.Debug("commit block", "number", bc.Number, "vs number", len(vs))
	p.receiveBlock(bc)
	for _, v := range vs {
		p.receiveVote(v)
	}
}

func (p *Wit) ExtraInfo() []byte {
	return []byte{}
}

func (p *Wit) CheckPackAuth(num uint64) bool {
	return true
}

func (p *Wit) StartDaemon(addr common.Address) {
	//打第一个区块
	log.Info("action")
}

//func (p *Wit) mainLoop() {
//	for {
//		select {
//		//case newBc := <-p.ProtoHandler.minerCh:
//		//	if p.lastNumber < newBc.Number {
//		//		p.sendBlock(newBc)
//		//	}
//		case v := <-p.ProtoHandler.resendvCh:
//			usedV, ok := p.usedVote[v.Number]
//			if ok {
//				p.ProtoHandler.SendVote(v.User, usedV)
//			}
//			p.receiveVote(v)
//		}
//	}
//}

func (p *Wit) packLoop() {
	p.packTimer = time.NewTimer(0)
	<-p.packTimer.C
	for {
		select {
		case <-p.packTimer.C:
			p.DoPackBlock()
		}
	}
}

func (p *Wit) PushBlock(block *block.Block) {
	p.receiveBlock(block)
}

//接受到区块的行为
func (p *Wit) receiveBlock(bc *block.Block) {
	//if bc.Number < p.lastNumber && p.extends(p.lastBlock, bc) {
	//	log.Debug("write last block", "number", bc.Number, "lst", p.lastNumber)
	//	//todo 如果符合要求，则直接存储
	//	p.writeBlock(bc)
	//	//也要检查父区块
	//	if !hasParent {
	//		p.ProtoHandler.RequestBlock(name, bc.ParentHash)
	//	}
	//	return
	//}
	p.pendingBlock.Store(bc.Hash, bc)

	//收集区块到branch中
	br, ok := p.branches[bc.Number]
	if !ok {
		br = NewBranch(p.cfg.Tolerance(), bc.Number, p.GetBlock)
		p.branches[bc.Number] = br
	}
	//if hasParent {
	b1, b2, ch := br.AddBlock(bc)
	p.checkBlock(b1, b2, bc.Number, ch)
	//} else {
	//	p.checkBlock(br.AddPendingBlock(bc))
	//}
	log.Debug("sendvote", "leafs", len(br.leafs), "number", bc.Number) //, "lstNumber", p.lastNumber)
	if len(br.leafs) == 1 {                                            //&& bc.Number == p.lastNumber+1 {
		log.Debug("addPendingBlock sendVote")
		p.sendVote(bc)
	}
	//如果没有父区块，收集完后获取父区块
	//if !hasParent {
	//	p.ProtoHandler.RequestBlock(name, bc.ParentHash)
	//	p.tempNode[bc.ParentHash] = br
	//	return
	//}
	//如果有子区块,循环解出pending关系
	//hash := bc.Hash
	//pendingBr, ok := p.tempNode[hash]
	//for ok {
	//	delete(p.tempNode, hash)
	//	hash = pendingBr.RemovePendingBlock(hash)
	//	pendingBr, ok = p.tempNode[hash]
	//}
}

func (p *Wit) extends(son, father *block.Block) bool {
	for son != nil && son.Number < father.Number {
		if son.ParentHash == father.ParentHash {
			return true
		}
		son = p.GetBlock(son.ParentHash)
	}
	return false
}

//接收到投票的行为
func (p *Wit) receiveVote(v *block.Vote) {
	//if v.Number < p.lastNumber {
	//	return
	//}
	br, ok := p.branches[v.Number]
	if !ok {
		br = NewBranch(p.cfg.Tolerance(), v.Number, p.GetBlock)
		p.branches[v.Number] = br
	}
	b1, b2, comfirmHash := br.AddVote(v)
	p.checkBlock(b1, b2, v.Number, comfirmHash)
}

//已经确认的区块，和可以作为父区块打包的区块
func (p *Wit) checkBlock(ensureBlock *block.Block, parentBlock *block.Block, num uint64, ch common.Hash) {
	if ensureBlock == nil || parentBlock == nil {
		return
	}
	//处理确认和打包
	log.Debug("check block", "ensureBlock", ensureBlock.Number, "parentBlock", parentBlock.Number)
	if ensureBlock.Hash != p.lastBlock.Hash {
		p.cftEnsure(ensureBlock, num, ch)
	}
	p.packBlock(parentBlock, false)
	//if br, ok := p.branches[ensureBlock.Number+1]; ok {
	//	for _, v := range br.leafs {
	//		if v.b == nil {
	//			continue
	//		}
	//		p.sendVote(v.b)
	//		return
	//	}
	//}
}

//等待两回合再行写入
//如果第三回合确认以后，还继续收集第三回合的投票，那么第三回合很有可能改变确认的目标（变得更高）
//所以我们直接简化逻辑，只允许当前产生更新，不在追究之前的
func (p *Wit) cftEnsure(bc *block.Block, num uint64, ch common.Hash) {
	if p.roundOneNumber > num {
		return
	}
	//循环确认
	comfirm := func(pbc *block.Block) {
		p.ConsensusProxy.OnBlockCFTConfirm(pbc)
		if pbc.ParentHash != p.lastBlock.Hash {
			pbc := p.GetBlock(pbc.ParentHash)
			if pbc != nil {
				p.ConsensusProxy.OnBlockCFTConfirm(pbc)
			}
		}
	}

	if p.roundOneNumber == num {
		//高度比当前的高
		if p.roundOne == nil || bc.Number > p.roundOne.Number {
			p.roundOne = bc
			comfirm(bc)
			//记录最后确认的区块
			p.lastCFTBlockHash = bc.Hash
		}
		return
	}
	comfirm(bc)
	//记录最后确认的区块
	p.lastCFTBlockHash = bc.Hash
	p.roundOneNumber = num
	//更新
	if p.roundTwo != nil {
		log.Debug("round two write", "roundTwo", p.roundTwo.Number)
		p.writeBlock(p.roundTwo)
	}
	if p.roundOne != nil {
		log.Debug("r1", "hash", p.roundOne.Hash, "ch", ch)
	}

	if p.roundOne != nil && p.roundOne.Hash == ch {
		log.Debug("round one write", "roundOne", p.roundOne.Number)
		p.writeBlock(p.roundOne)
		p.roundOne = nil
		p.roundTwo = nil
	} else {
		p.roundTwo = p.roundOne
		if p.roundOne == nil || bc.Hash != p.roundOne.Hash {
			p.roundOne = bc
		} else {
			//当没有继续确认时，则不需要更新
			p.roundOne = nil
		}
	}
}

//确认区块
func (p *Wit) writeBlock(bc *block.Block) {
	//if bc.Number == 10 {
	//	panic("stop")
	//}
	//如果当前已确定的高度高于投票高度，放弃写入区块
	//if bc.Number <= p.lastNumber {
	//	return	//}
	//继续获取父区块
	if bc.Hash == p.lastBlock.Hash {
		return
	}
	if bc.ParentHash != p.lastBlock.Hash {
		log.Debug("write parent", "number", bc.Number, "parent", bc.ParentHash, "last", p.lastBlock.Number, "lastHash", p.lastBlock.Hash)
		pbc := p.GetBlock(bc.ParentHash)
		if pbc != nil {
			p.writeBlock(pbc)
		}
	}
	p.lastBlock = bc
	//p.lastNumber = bc.Number
	//存储数据
	//if bc.Number%10000 == 0 {
	log.Debug("writeBlock", "number", bc.Number, "hash", bc.Hash, "parentHash", bc.ParentHash, "payload", len(bc.Payload))
	//}
	//删掉对应的高度
	p.deleteBranch(p.branches[bc.Number], bc.Number)
	p.OnBlockConfirm(bc)
}

//打包区块
func (p *Wit) packBlock(bc *block.Block, first bool) {
	//不允许再次打包
	if p.lastEnsureBlock != nil && p.lastEnsureBlock.Number >= bc.Number {
		return
	}
	p.lastEnsureBlock = bc

	p.OnBlockAvailable(bc)
	//如果已经有区块则直接进行投票
	//if blockMap, ok := p.pendingNumberBlock[bc.Number+1]; ok {
	//	for _, v := range blockMap {
	//		p.sendVote(v)
	//		return
	//	}
	//}
	//如果没有，判断自己是不是打包者
	if (!p.cfg.Rotation() && p.cfg.Turn() == 0) || (p.cfg.Rotation() && int(bc.Number+1)%p.cfg.SaintLen() == int(p.cfg.Turn())) {
		p.DoPackBlock()
		return
	}
	//第一个包只允许当轮者打包
	if first {
		log.Debug("not the first packer", "number", bc.Number, "turn", p.cfg.Turn())
		return
	}
	p.lastEnsureBlock = bc
	//如果不是打包者，判断自己是否有打包权，有则等待不定时间进行出块
	x := int(p.cfg.Turn()) - int(bc.Number+1)%p.cfg.SaintLen()
	if x < 0 {
		x = int(p.cfg.Turn()) + p.cfg.SaintLen() - int(bc.Number+1)%p.cfg.SaintLen()
	}
	p.packTimer.Reset(time.Duration(consensus_service.ViewChangeDuration.Milliseconds()+int64(x)*100) * time.Millisecond)
}

//
//func (p *Wit) sendBlock(bc *block.Block) {
//	p.ProtoHandler.BroadcastBlock(bc)
//	p.receiveBlock(common.EmptyAddress, bc)
//}

func (p *Wit) sendVote(bc *block.Block) {
	if bc.Number <= p.lastVoteNumber {
		log.Debug("sendVote but number to low", "number", bc.Number, "lst", p.lastVoteNumber)
		return
	}
	p.lastVoteNumber = bc.Number
	v := &block.Vote{
		User:        p.cfg.Coinbase(),
		ConfirmHash: p.lastCFTBlockHash,
		Number:      bc.Number,
		BC:          bc.Hash,
	}
	sig, _ := p.Signature(v.RlpHash())
	v.SetSig(sig)
	log.Debug("vote", "bc", bc.Hash)
	p.SynchronizeRun(func() { p.receiveVote(v) })
	p.usedVote[bc.Number] = v
	p.OnVote(v)
	//不断发送签名
	//p.reSendVote(v, bc.Number)
}

func (p *Wit) reSendVote(v *block.Vote, number uint64) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Debug("resendVote", "number", number) //, "lst", p.lastNumber)
		//if p.lastNumber+1 == number {
		p.OnVote(v)
		p.reSendVote(v, number)
		//}
	}()
}

func (p *Wit) GetBlock(hash common.Hash) *block.Block {
	v, ok := p.pendingBlock.Load(hash)
	if ok {
		bc := v.(*block.Block)
		return bc
	}
	//不可能引用一个不被确认的块，如果是则直接返回没有
	if p.lastBlock.Hash == hash {
		return p.lastBlock
	}
	return nil
}

//删除一个分支，主要原因可能是分支完成确认
func (p *Wit) deleteBranch(br *branch, number uint64) {
	if br == nil {
		return
	}
	delete(p.branches, number)
	//for k, _ := range p.pendingNumberBlock[number] {
	//	delete(p.pendingBlock, k)
	//}
	//delete(p.pendingNumberBlock, number)
}

//高度所对应的分叉结构
type branch struct {
	height      uint64
	f21         int
	collectVote map[common.Address]*block.Vote //收集到的投票
	wait        map[common.Hash]struct{}       //等待的区块，这里也需要判断，避免有投票无区块
	//pending     map[common.Hash]common.Hash    //引用区块未确定，这里需要等待引用的区块
	leafs    map[common.Hash]*node //已经具有完备区块的节点
	treeRoot *TreeRoot             //当前投票树
	getBlock func(hash common.Hash) *block.Block
}

type node struct {
	b       *block.Block          //当前区块
	votelen int                   //投票总数
	sonNode map[common.Hash]*node //所有子结点
	voteSet mapset.Set            //所有投票
}

func (n *node) String() string {
	return fmt.Sprintf("[%v,%s -> %s : %v , %v]", n.b.Number, n.b.Hash.String(), n.b.ParentHash.String(), n.votelen, n.voteSet.Cardinality())
}

type TreeRoot struct {
	root *node
	Node map[uint64]map[common.Hash]*node //所有节点，key是高度，第二层key是结点区块hash
}

func (tr *TreeRoot) String() string {
	return fmt.Sprintf("%v", tr.Node)
}

func NewBranch(f21 int, height uint64, getBlock func(hash common.Hash) *block.Block) *branch {
	return &branch{
		height:      height,
		f21:         f21,
		collectVote: make(map[common.Address]*block.Vote),
		wait:        make(map[common.Hash]struct{}),
		//pending:     make(map[common.Hash]common.Hash),
		leafs:    make(map[common.Hash]*node),
		treeRoot: nil,
		getBlock: getBlock,
	}
}

//当一个块的引用区块收集齐了，那么重新进行build
//func (p *branch) RemovePendingBlock(parentHash common.Hash) (hash common.Hash) {
//	hash = p.pending[parentHash]
//	delete(p.pending, parentHash)
//	if len(p.collectVote) > p.f21 && len(p.pending) == 0 {
//		p.buildTree()
//	}
//	return
//}

//增加一个有未获取到引用区块的块
//func (p *branch) AddPendingBlock(bc *block.Block) (ensureBlock *block.Block, parentBlock *block.Block) {
//	p.pending[bc.ParentHash] = bc.Hash
//	return p.AddBlock(bc)
//}

func (p *branch) AddBlock(bc *block.Block) (ensureBlock *block.Block, parentBlock *block.Block, hash common.Hash) {
	delete(p.wait, bc.Hash)
	n, ok := p.leafs[bc.Hash]
	if !ok {
		n = &node{
			b:       bc,
			votelen: 0,
			sonNode: nil,             //所有子结点
			voteSet: mapset.NewSet(), //所有投票
		}
		p.leafs[bc.Hash] = n
	} else {
		n.b = bc
	}
	log.Debug("AddBlock", "bc", n.b.Hash, "Number", n.b.Number, "collect", n.voteSet.Cardinality())
	//单个区块满足了，那么返回上面，上面对低于他的块，且是待定的块都会直接确定，并会继续进行区块的出块
	if n.voteSet.Cardinality() > p.f21 {
		confirmHash := findDuplicateConfirmHash(p.collectVote, p.f21+1)
		return n.b, n.b, confirmHash
	}
	return nil, nil, common.Hash{}
}

func (p *branch) AddVote(v *block.Vote) (ensureBlock *block.Block, parentBlock *block.Block, confirmHash common.Hash) {
	if p.collectVote[v.User] != nil {
		return nil, nil, common.Hash{}
	}
	p.collectVote[v.User] = v

	//todo 判断所有的vote是不是指向同一个
	if len(p.collectVote) > p.f21 {
		confirmHash = findDuplicateConfirmHash(p.collectVote, p.f21+1)
	}

	n, ok := p.leafs[v.BC]
	if !ok {
		n = &node{
			b:       nil,
			votelen: 0,
			sonNode: nil,             //所有子结点
			voteSet: mapset.NewSet(), //所有投票
		}
		p.wait[v.BC] = struct{}{}
		p.leafs[v.BC] = n
	}
	n.votelen += 1
	n.voteSet.Add(v)
	log.Debug("addVote", "user", v.User, "bc", v.BC, "Number", v.Number, "bcExist", n.b == nil, "collect", n.voteSet.Cardinality())
	//单个区块满足了,那就直接入库，然后慢慢补块
	if n.b != nil && n.voteSet.Cardinality() > p.f21 {
		return n.b, n.b, confirmHash
	}
	//如果不是单个，那么必须等收集到足够的引用块
	if len(p.collectVote) > p.f21 && len(p.wait) == 0 {
		b1, b2 := p.buildTree()
		return b1, b2, confirmHash
	}
	return nil, nil, common.Hash{}
}

func findDuplicateConfirmHash(collectVote map[common.Address]*block.Vote, n int) common.Hash {
	hashCount := make(map[common.Hash]int)

	// 遍历 collectVote
	for _, v := range collectVote {
		// 记录 ConfirmHash 出现的次数
		hashCount[v.ConfirmHash]++
		// 判断是否达到 n 次相同的 ConfirmHash
		if hashCount[v.ConfirmHash] == n {
			// 输出满足条件的 ConfirmHash
			return v.ConfirmHash
		}
	}
	return common.Hash{}
}

//构造当前高度的树，返回确定的区块和下一个出块的区块
func (p *branch) buildTree() (ensureBlock *block.Block, parentBlock *block.Block) {
	p.treeRoot = &TreeRoot{
		Node: make(map[uint64]map[common.Hash]*node),
	}
	p.treeRoot.Node[p.height] = make(map[common.Hash]*node)
	//构成叶子节点
	for _, n := range p.leafs {
		p.treeRoot.Node[p.height][n.b.Hash] = n
	}
	return p.recursionBuildTree(p.height)
}

//循环构造树
func (p *branch) recursionBuildTree(height uint64) (ensureBlock *block.Block, parentBlock *block.Block) {
	log.Debug("recursionBuildTree", "height", height)
	for {
		//对当前高度下所有节点寻找父节点
		for _, v := range p.treeRoot.Node[height] {
			bc := p.getBlock(v.b.ParentHash)
			if bc == nil {
				log.Error("get block fail", "num", v.b.Number, "parentHash", v.b.ParentHash)
				panic(fmt.Errorf("block fail"))
			}
			layer, ok := p.treeRoot.Node[height-1]
			if !ok {
				layer = make(map[common.Hash]*node)
				p.treeRoot.Node[height-1] = layer
			}
			log.Debug("build tree bc", "number", height-1, "hash", bc == nil)
			log.Debug("build tree tr before", "value", p.treeRoot.String())
			n, ok := layer[bc.Hash]
			if !ok {
				n = &node{
					b:       bc,
					votelen: 0,
					sonNode: make(map[common.Hash]*node), //所有子结点
					voteSet: mapset.NewSet(),             //所有投票
				}
			}
			n.votelen += v.voteSet.Cardinality()
			n.sonNode[v.b.Hash] = v
			n.voteSet = n.voteSet.Union(v.voteSet)
			p.treeRoot.Node[height-1][bc.Hash] = n
			log.Debug("build tree tr", "value", p.treeRoot.String())
			if n.voteSet.Cardinality() > p.f21 {
				p.treeRoot.root = n
				break
			}
		}
		//找到根，则树完成
		if p.treeRoot.root != nil {
			break
		} else {
			height--
		}
	}
	n := p.treeRoot.root
	//不断寻找投票数更多的子节点，直到找到叶子节点
	for {
		if n.sonNode == nil {
			return p.treeRoot.root.b, n.b
		}
		var maxNode *node
		var votelen int = 0
		for _, son := range n.sonNode {
			if son.votelen > votelen {
				maxNode = son
				votelen = son.votelen
			}
		}
		log.Debug("maxNode", "bc", maxNode.b.Number, "hash", maxNode.b.Hash)
		n = maxNode
	}
}
