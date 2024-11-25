package core

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"sync"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/zerror"
	"witCon/log"
	"witCon/stat"
)

// 因为只考虑区块会按顺序写入，所以很多操作没有处理
// 如果区块不按顺序写入，会出错
type WorldState struct {
	state    []map[common.Address][]byte
	current  uint64
	dirty    map[common.Hash]*State //未确定的区块的状态
	dirtyNum map[uint64][]common.Hash
	lock     sync.RWMutex
}

type State struct {
	ws     *WorldState
	parent common.Hash //缓存父状态
	state  []map[common.Address][]byte
}

func NewState(ws *WorldState, parent common.Hash) *State {
	st := &State{
		ws:     ws,
		parent: parent,
		state:  make([]map[common.Address][]byte, common.InitDirectShardCount),
	}
	for i := 0; i < common.InitDirectShardCount; i++ {
		st.state[i] = make(map[common.Address][]byte)
	}
	return st
}

// 读取余额，如果读取不到就从父状态中读取，如果没有父状态，就从世界状态中读取，并缓存
func (s *State) getState(addr common.Address) []byte {
	shard := common.Shard(addr)
	shardState := s.state[shard]
	if v, ok := shardState[addr]; ok {
		return v
	}
	parentState := s.ws.dirty[s.parent]
	if parentState != nil {
		v := parentState.getState(addr)
		//s.state[addr] = v
		return v
	}
	wsShardState := s.ws.state[shard]
	if v, ok := wsShardState[addr]; ok {
		//s.state[addr] = v
		return v
	} else {
		//s.state[addr] = nil
		return nil
	}
}

func (s *State) WriteState(addr common.Address, value []byte) {
	shard := common.Shard(addr)
	shardState := s.state[shard]
	shardState[addr] = value
}

func NewWorldState() *WorldState {
	ws := &WorldState{
		state:    make([]map[common.Address][]byte, common.InitDirectShardCount),
		current:  0,
		dirty:    make(map[common.Hash]*State),
		dirtyNum: make(map[uint64][]common.Hash),
	}

	for i := 0; i < common.InitDirectShardCount; i++ {
		ws.state[i] = make(map[common.Address][]byte)
	}
	return ws
}

func (ws *WorldState) GetState(currentHash common.Hash, addr common.Address) []byte {
	if v, ok := ws.dirty[currentHash]; ok {
		return v.getState(addr)
	}
	return nil
}

func (ws *WorldState) OnBlockConfirm(bc *block.Block) {
	go func() {
		ws.lock.Lock()
		defer ws.lock.Unlock()
		ws.current = bc.Number
		dirty := ws.dirty[bc.Hash]
		//清理缓存
		for _, hash := range ws.dirtyNum[bc.Number] {
			delete(ws.dirty, hash)
		}
		delete(ws.dirtyNum, bc.Number)
		if dirty != nil {
			for i, m := range dirty.state {
				for k, v := range m {
					ws.state[i][k] = v
					//log.Debug("write state", "i", i, "key", k, "value", string(v))
				}
			}
			delete(ws.dirty, bc.Hash)
		}
	}()
}

func (ws *WorldState) ExecuteBcTime(bc *block.Block, txs []*block.Transaction) ([]*block.ShardBody, common.Hash, int64, int64, error) {
	t1 := time.Now().UnixMicro()
	vm, err := ws.PreExecuteBc(bc, txs)
	t2 := time.Now().UnixMicro()
	if err != nil {
		return nil, [32]byte{}, 0, 0, err
	}
	t3 := time.Now().UnixMicro()
	a, b := ws.CommitRoot(vm)
	t4 := time.Now().UnixMicro()
	fmt.Println(fmt.Sprintf("exe bc, t1:%v,t2:%v", t2-t1, t4-t3))
	return a, b, t2 - t1, t4 - t3, nil
}

func (ws *WorldState) ExecuteBc(bc *block.Block, txs []*block.Transaction) ([]*block.ShardBody, common.Hash, error) {
	//t1 := time.Now().UnixMicro()
	vm, err := ws.PreExecuteBc(bc, txs)

	stat.DBlockTimeTrace.AddDBTime(bc.Number, len(txs), "PreExecuteBc")
	//t2 := time.Now().UnixMicro()
	if err != nil {
		return nil, [32]byte{}, err
	}
	//t3 := time.Now().UnixMicro()
	a, b := ws.CommitRoot(vm)
	//t4 := time.Now().UnixMicro()
	//fmt.Println(fmt.Sprintf("exe bc, t1:%v,t2:%v", t2-t1, t4-t3))
	return a, b, nil
}

func (ws *WorldState) PreExecuteBc(bc *block.Block, txs []*block.Transaction) (*VirtualMachine, error) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	if bc.Number <= ws.current {
		return nil, zerror.New(fmt.Sprintf("fail number %d <= %d", bc.Number, ws.current), "", 2001)
	}
	st := NewState(ws, bc.ParentHash)
	vm := NewVirtualMachine()
	err := vm.ExecuteTxs(st, txs)
	if err != nil {
		return nil, err
	}
	ws.SetDirtyState(bc.Hash, bc.Number, st)
	return vm, nil
}

func (ws *WorldState) CommitRoot(vm *VirtualMachine) ([]*block.ShardBody, common.Hash) {
	sbl, root := vm.GenerateShardBody()
	return sbl, root
}

func (ws *WorldState) SetDirtyState(hash common.Hash, number uint64, st *State) {
	ws.dirty[hash] = st
	//把num记上，后面会清除掉
	if lst, ok := ws.dirtyNum[number]; ok {
		ws.dirtyNum[number] = append(lst, hash)
	} else {
		lstN := make([]common.Hash, 1)
		lstN[0] = hash
		ws.dirtyNum[number] = lstN
	}
}

func (ws *WorldState) VerifyShardMulti(bc *block.Block, bodies []*block.ShardBody, pool *ants.Pool) error {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	st := NewState(ws, bc.ParentHash)
	var wg sync.WaitGroup
	var err error
	for index, shardBody := range bodies {
		_index := index
		_shardBody := shardBody
		wg.Add(1)
		poolErr := pool.Submit(func() {
			defer wg.Done()
			vm := NewShardVirtualMachine(st, _shardBody, uint16(_index))
			shardErr := vm.ExecuteTxs(st, _shardBody.Txs, _shardBody.TxIndex, bc.LedgerHash)
			if shardErr != nil {
				err = shardErr
			}
		})
		if poolErr != nil {
			err = poolErr
		}
	}
	wg.Wait()
	ws.SetDirtyState(bc.Hash, bc.Number, st)
	if err != nil {
		log.Error("verify shard fail", "err", err)
		return err
	}
	return nil
}

func (ws *WorldState) VerifyShard(bc *block.Block, body *block.ShardBody, shardIndex uint16) error {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	st := NewState(ws, bc.ParentHash)
	vm := NewShardVirtualMachine(st, body, shardIndex)
	return vm.ExecuteTxs(st, body.Txs, body.TxIndex, bc.LedgerHash)
}
