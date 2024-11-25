package core

import (
	"time"
	"witCon/chaincode"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/common/zerror"
	"witCon/crypto"
	"witCon/log"
)

type VirtualMachine struct {
	shardState  []*ShardState //各分片需要验证的状态
	st          *State
	shardBodies []*block.ShardBody
	//各分片需要验证的交易
	currentIndex int
	currentShard uint16
}

func NewVirtualMachine() *VirtualMachine {
	vm := &VirtualMachine{
		shardState:   make([]*ShardState, common.InitDirectShardCount),
		shardBodies:  make([]*block.ShardBody, common.InitDirectShardCount),
		currentIndex: 0,
		currentShard: 0,
	}
	for i := 0; i < common.InitDirectShardCount; i++ {
		vm.shardBodies[i] = new(block.ShardBody)
	}
	return vm
}

func (vm *VirtualMachine) ExecuteTxs(st *State, txs []*block.Transaction) error {
	vm.st = st
	for index, tx := range txs {
		vm.currentIndex = index
		_s := common.Shard(tx.From)
		vm.currentShard = _s
		err := chaincode.ResolveTx(tx, vm)
		if err != nil {
			return err
		}
		//把交易分片下去
		sb := vm.shardBodies[_s]
		if sb.Txs == nil {
			sb.Txs = make([]*block.Transaction, 1)
			sb.TxIndex = make([]uint16, 1)
			sb.Txs[0] = tx
			sb.TxIndex[0] = uint16(index)
		} else {
			sb.Txs = append(sb.Txs, tx)
			sb.TxIndex = append(sb.TxIndex, uint16(index))
		}
	}

	return nil
}

func (vm *VirtualMachine) ReadState(key common.Address) []byte {
	_s, s := vm.getShard(key)
	value := vm.st.getState(key)
	readOp := &block.StateOp{
		Index:   uint16(vm.currentIndex),
		Read:    true,
		InShard: vm.currentShard,
		Key:     key,
		Value:   value,
	}
	s.AddState(readOp)
	if _s != vm.currentShard {
		_, selfShard := vm.getShardByS(vm.currentShard)
		selfShard.AddOp(readOp)
	}
	return value
}

// opList记录得是别人读我得，state记录得是我读所有得
func (vm *VirtualMachine) WriteState(key common.Address, value []byte) {
	_s, s := vm.getShard(key)
	vm.st.WriteState(key, value)
	writeOp := &block.StateOp{
		Index:   uint16(vm.currentIndex),
		Read:    false,
		InShard: vm.currentShard,
		Key:     key,
		Value:   value,
	}
	s.AddState(writeOp)
	if _s != vm.currentShard {
		_, selfShard := vm.getShardByS(vm.currentShard)
		selfShard.AddOp(writeOp)
	}
}

func (vm *VirtualMachine) getShard(key common.Address) (uint16, *ShardState) {
	_s := common.Shard(key)
	return vm.getShardByS(_s)
}

func (vm *VirtualMachine) getShardByS(_s uint16) (uint16, *ShardState) {
	s := vm.shardState[_s]
	if s == nil {
		s = &ShardState{
			OpList:    make([]*block.StateOp, 0, 1),
			StateList: make([]*block.StateOp, 0, 1),
		}
		vm.shardState[_s] = s
	}
	return _s, s
}

// 生成这棵树，和分片数相关，如果分片很多，会非常的耗时。
// 64个分片的时候，基本还可以承受住
func (vm *VirtualMachine) GenerateShardBody() ([]*block.ShardBody, common.Hash) {
	shardRootList := make([]common.Hash, common.InitDirectShardCount)

	ct := int64(0)
	cpt := int64(0)
	for shardIndex, ss := range vm.shardState {
		if ss == nil {
			//rlp编码不可以为空
			for i := 0; i < common.InitDirectShardCount; i++ {
				sp := vm.shardBodies[i].ShardProof
				if sp == nil {
					sp = make([]*crypto.MerkleTreeProof, common.InitDirectShardCount)
					vm.shardBodies[i].ShardProof = sp
				}
				sp[shardIndex] = &crypto.MerkleTreeProof{}
			}
			continue
		}
		//log.Debug("shardIndex", "index", shardIndex, "shardState", ss.String())
		vm.shardBodies[shardIndex].SelfOp = ss.SelfOP(uint16(shardIndex))
		vm.shardBodies[shardIndex].ReadState = ss.ReadState(uint16(shardIndex))
		ct1 := time.Now().UnixMicro()
		smt := ss.GetMerkleTree()
		ct2 := time.Now().UnixMicro()
		ct = ct2 - ct1 + ct
		//fmt.Println(fmt.Sprintf("shard %v mt", shardIndex))
		//smt.Print()
		//为其他分片，生成自己分片的路径
		for i := 0; i < common.InitDirectShardCount; i++ {
			sp := vm.shardBodies[i].ShardProof
			if sp == nil {
				sp = make([]*crypto.MerkleTreeProof, common.InitDirectShardCount)
				vm.shardBodies[i].ShardProof = sp
			}
			ct3 := time.Now().UnixMicro()
			if i != shardIndex {
				p := smt.MakeProof(uint16(i))
				if len(p.Proof) == 1 && common.MultiMerkle {
					vm.shardBodies[i].ShardProof[shardIndex] = &crypto.MerkleTreeProof{Nil: true}
				} else {
					vm.shardBodies[i].ShardProof[shardIndex] = p
				}
			} else {
				vm.shardBodies[i].ShardProof[shardIndex] = &crypto.MerkleTreeProof{}
			}
			ct4 := time.Now().UnixMicro()
			cpt = ct4 - ct3 + cpt
		}
		shardRootList[shardIndex] = smt.GetRoot()
	}
	//wg.Wait()
	//fmt.Println("ct time is", ct, cpt)

	//for index, sb := range vm.shardBodies {
	//	log.Debug("----------sb", "index", index)
	//	//sb.Print()
	//}

	//生成第二层的merkle树
	mt := &crypto.MerkleTree{}
	mt.MakeTree(shardRootList)
	stateRoot := mt.GetRoot()
	if common.MultiMerkle {
		for _, sb := range vm.shardBodies {
			shardExist := make([]bool, common.InitDirectShardCount)
			for i, p := range sb.ShardProof {
				if !p.Nil {
					shardExist[i] = true
				} else {
					shardExist[i] = false
				}
			}
			mp := mt.MakeMultiProof(shardExist)
			sb.MultiProof = mp
		}
	} //mt.Print()
	return vm.shardBodies, stateRoot
}

type ShardVirtualMachine struct {
	st *State
	//ss           *ShardState
	readState [][]byte //这个是需要别人给的
	offset    int      //计算read state 的index

	shardProof   []*crypto.MerkleTreeProof //这个是别人shard的证明，这个也需要提供
	mProof       *crypto.MultiMerkleProof  //简化的多重mp
	localOP      []*block.StateOp          //这个是别人写得顺序，这个也需要提供
	selfOpOffset int                       //计算Op执行情况的offset

	crossOP [][]*block.StateOp //这个是自己生成的

	shardIndex   uint16
	currentIndex uint16
}

func NewShardVirtualMachine(st *State, shardBody *block.ShardBody, shardIndex uint16) *ShardVirtualMachine {
	return &ShardVirtualMachine{
		st:         st,
		readState:  shardBody.ReadState,
		shardProof: shardBody.ShardProof,
		mProof:     shardBody.MultiProof,
		localOP:    shardBody.SelfOp,
		crossOP:    make([][]*block.StateOp, common.InitDirectShardCount),
		shardIndex: shardIndex,
	}
}

// 返回运行完，获取到的状态根
func (vm *ShardVirtualMachine) ExecuteTxs(st *State, txs []*block.Transaction, txIndex []uint16, stateRoot common.Hash) error {
	vm.st = st
	selfOpSize := len(vm.localOP)
	var localOP *block.StateOp
	if selfOpSize > 0 {
		localOP = vm.localOP[vm.selfOpOffset]
	}
	doLocalOP := func() {
		if localOP.Read {
			//log.Debug("get", "key", localOP.Key, "shard", common.Shard(localOP.Key))
			_v := vm.st.getState(localOP.Key)
			localOP.Value = _v
		} else {
			//log.Debug("write", "key", localOP.Key, "shard", common.Shard(localOP.Key), "value", string(localOP.Value))
			vm.st.WriteState(localOP.Key, localOP.Value)
		}
		vm.selfOpOffset++
		//如果超过了操作大小，就可以结束了
		if vm.selfOpOffset >= selfOpSize {
			localOP = nil
		} else {
			localOP = vm.localOP[vm.selfOpOffset]
		}
	}

	for index, tx := range txs {
		//先判断别人的读写是否更早
		for localOP != nil && localOP.Index < txIndex[index] {
			doLocalOP()
		}
		vm.currentIndex = txIndex[index]
		err := chaincode.ResolveTx(tx, vm)
		if err != nil {
			return err
		}
	}

	//清空localOP
	for localOP != nil {
		doLocalOP()
	}
	var currentRoot common.Hash
	if common.MultiMerkle {
		currentRoot = vm.getMultiBlockRoot(vm.mProof, stateRoot)
	} else {
		currentRoot = vm.getBlockRoot()
	}
	if stateRoot != currentRoot {
		log.Error("err shard state", "stateRoot", stateRoot, "currentRoot", currentRoot)
		return ErrShardState
	}
	return nil
}

// 其实这里可以看出来，index很重要，但是key可以没有，我们可以凑出来
func (vm *ShardVirtualMachine) ReadState(key common.Address) []byte {
	_s := common.Shard(key)
	var value []byte
	if vm.shardIndex != _s {
		value = vm.readState[vm.offset]
		vm.offset++
	} else {
		//log.Debug("get", "key", key, "shard", common.Shard(key), "shardIndex", vm.shardIndex)
		value = vm.st.getState(key)
	}
	readOp := &block.StateOp{
		Index:   vm.currentIndex,
		InShard: vm.shardIndex,
		Read:    true,
		Key:     key,
		Value:   value,
	}
	scd := vm.crossOP[_s] //往对应跨合约写入中插入数据
	if scd == nil {
		scd = make([]*block.StateOp, 1)
		scd[0] = readOp
	} else {
		scd = append(scd, readOp)
	}
	vm.crossOP[_s] = scd

	return value
}

// 其实这里可以看出来，实际上key和value都可以没有，我们都可以凑出来
func (vm *ShardVirtualMachine) WriteState(key common.Address, value []byte) {
	_s := common.Shard(key)
	writeOp := &block.StateOp{
		Index:   vm.currentIndex,
		InShard: vm.shardIndex,
		Read:    false,
		Key:     key,
		Value:   value,
	}
	if vm.shardIndex == _s {
		//log.Debug("write", "key", key, "shard", common.Shard(key), "shardIndex", vm.shardIndex)
		vm.st.WriteState(key, value)
	}
	scd := vm.crossOP[_s] //往对应跨合约写入中插入数据
	if scd == nil {
		scd = make([]*block.StateOp, 1)
		scd[0] = writeOp
	} else {
		scd = append(scd, writeOp)
	}
	vm.crossOP[_s] = scd
	return
}

// 首先，本地的交易运行一遍，没有问题。
// 然后，本地的状态运行一遍，也没有问题。
// 最后根据运行结果，生成根，也没有问题。
// 那么就没有问题。
func (vm *ShardVirtualMachine) getBlockRoot() common.Hash {
	//配合shardCrossData 和 shardProof,可以获取到别人的shard的树
	//配合shardCrossData中自己的数据，和selfOp可以获取到自己的shard的树
	//对所有分片的树取merkle树，就可以获取到最终的树
	//验证最终的树，就可以获取到结果
	shardRootList := make([]common.Hash, common.InitDirectShardCount)
	for shardIndex, scd := range vm.crossOP {
		if uint16(shardIndex) == vm.shardIndex {
			if len(vm.localOP) == 0 && len(scd) == 0 {
				continue
			}
			sol := make([][]*block.StateOp, 2)
			sol[0] = vm.localOP
			sol[1] = scd
			mt := ToCrossShardMerkleRoot(sol)
			root := mt.GetRoot()
			shardRootList[shardIndex] = root

			//opLen := len(vm.localOP) + len(scd)
			//log.Debug(fmt.Sprintf("verify shardIndex %v ", shardIndex))
			//log.Debug(fmt.Sprintf("verify shardMt %v, len: %v", root.String(), opLen))
			mt.Print()
		} else {
			var hash = common.EmptyHash
			if scd != nil {
				data, err := rlp.EncodeToBytes(scd)
				if err != nil {
					log.Crit("fail encode", "err", err)
				}

				hash = crypto.Sha256(data)
			}
			if vm.shardProof == nil {
				shardRootList[shardIndex] = hash
				continue
			}
			proof := vm.shardProof[shardIndex]
			if proof.Nil {
				shardRootList[shardIndex] = hash
			} else {
				shardRoot := proof.VerifyProof(hash)
				shardRootList[shardIndex] = shardRoot
				//log.Debug(fmt.Sprintf("verify shardIndex %v ", shardIndex))
				//log.Debug(fmt.Sprintf("verify shardHash %v ", hash.String()))
				proof.Print()
				//log.Debug(fmt.Sprintf("verify shardRoot %v ", shardRoot.String()))
			}
		}
	}
	//fmt.Println("verify end")
	mt := &crypto.MerkleTree{}
	mt.MakeTree(shardRootList)
	stateRoot := mt.GetRoot()
	return stateRoot
}

func (vm *ShardVirtualMachine) getMultiBlockRoot(mproof *crypto.MultiMerkleProof, stateRoot common.Hash) common.Hash {
	//配合shardCrossData 和 shardProof,可以获取到别人的shard的树
	//配合shardCrossData中自己的数据，和selfOp可以获取到自己的shard的树
	//对所有分片的树取merkle树，就可以获取到最终的树
	//验证最终的树，就可以获取到结果
	shardRootList := make([]common.Hash, 0)
	for shardIndex, scd := range vm.crossOP {
		if uint16(shardIndex) == vm.shardIndex {
			if len(vm.localOP) == 0 && len(scd) == 0 {
				shardRootList = append(shardRootList, common.EmptyHash)
				continue
			}
			sol := make([][]*block.StateOp, 2)
			sol[0] = vm.localOP
			sol[1] = scd
			mt := ToCrossShardMerkleRoot(sol)
			root := mt.GetRoot()
			shardRootList = append(shardRootList, root)
		} else {
			var hash = common.EmptyHash
			if scd != nil {
				data, err := rlp.EncodeToBytes(scd)
				if err != nil {
					log.Crit("fail encode", "err", err)
				}

				hash = crypto.Sha256(data)
			}
			if vm.shardProof == nil {
				shardRootList[shardIndex] = hash
				continue
			}
			proof := vm.shardProof[shardIndex]
			if !proof.Nil {
				shardRoot := proof.VerifyProof(hash)
				shardRootList = append(shardRootList, shardRoot)
			}
		}
	}
	currentRoot := mproof.VerifyProof(shardRootList)
	if stateRoot != currentRoot {
		log.Error("err shard state", "stateRoot", stateRoot, "currentRoot", currentRoot)
	}
	return currentRoot
}

var (
	ErrShardState = zerror.New("错误的分片证明", "fa", 1002)
)
