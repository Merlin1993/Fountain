package crypto

import (
	"fmt"
	"witCon/common"
	"witCon/log"
)

type MerkleTr interface {
	MakeProof(shard uint16) *MerkleTreeProof
	GetRoot() common.Hash
	Print()
}

// 针对稀疏树的优化
// 因为可能会有很多的空的节点，我们将空的节点给压缩掉
type CompressMerkleTree struct {
	MerkleTree
	SolidShardOffset []int
	Offset           int
	Hashes           []common.Hash
}

func (mt *CompressMerkleTree) WriteItem(hash common.Hash) {
	mt.Offset++
	if hash == common.EmptyHash {
		mt.SolidShardOffset = append(mt.SolidShardOffset, 0)
	} else {
		mt.Hashes = append(mt.Hashes, hash)
		mt.SolidShardOffset = append(mt.SolidShardOffset, len(mt.Hashes))
	}
}

func (mt *CompressMerkleTree) CommitTree() {
	mt.MerkleTree.MakeTree(mt.Hashes)
}

func (mt *CompressMerkleTree) MakeProof(shard uint16) *MerkleTreeProof {
	offset := mt.SolidShardOffset[int(shard)]
	if offset != 0 {
		return mt.MerkleTree.MakeProof(uint16(offset) - 1)
	} else {
		return &MerkleTreeProof{
			Proof: []common.Hash{mt.MerkleTree.GetRoot()},
		}
	}
}

// todo,只处理了偶数，没有处理奇数。好像不对！
// 即使你处理了第一层的单数问题，第二层第三层还是可能会单数
type MerkleTree struct {
	Layer [][]common.Hash //根据叶子到树根的高度排列，分别是1，2，3，4...，12，34...，1234
}

func (mt *MerkleTree) MakeTree(hashes []common.Hash) {
	size := len(hashes)
	var layer = 0
	mt.Layer = make([][]common.Hash, 1)
	mt.Layer[0] = hashes
	layer++
	two := make([]byte, 64)

	for size > 1 {
		size = 0
		var left = true
		mt.Layer = append(mt.Layer, make([]common.Hash, 0))
		for _, h := range hashes {
			if left {
				//先取左边
				copy(two[:32], h[:])
				left = false
			} else {
				//再取右边
				copy(two[32:], h[:])
				rhash := Sha256(two)
				mt.Layer[layer] = append(mt.Layer[layer], rhash)
				left = true
				size++
			}
		}
		//如果不是偶数，最后一个往上追加
		if len(hashes)%2 != 0 {
			copy(two[32:], common.EmptyHash[:])
			rhash := Sha256(two)
			mt.Layer[layer] = append(mt.Layer[layer], rhash)
			size++
		}

		hashes = mt.Layer[layer] //继续往下遍历
		layer++
	}
}

func (mt *MerkleTree) Print() {
	if !common.PrintMerkleTree {
		return
	}
	log.Debug("---------MT-------------")
	for index, hs := range mt.Layer {
		log.Debug(fmt.Sprintf("L%v ", index))
		for _, h := range hs {
			log.Debug(fmt.Sprintf("%v  ", h.String()))
		}
		log.Debug("")
	}
}

func (mt *MerkleTree) MakeProof(shard uint16) *MerkleTreeProof {
	if len(mt.Layer) == 0 {
		return nil
	}
	layer := make([]common.Hash, 0)
	left := make([]bool, 0)
	shardIndex := shard
	//layer = append(layer, mt.Layer[0][shardIndex])
	//最后一个值是root
	for _, l := range mt.Layer[0 : len(mt.Layer)-1] {
		//偶数取下一个奇数，奇数取上一个偶数
		if shardIndex%2 == 0 {
			left = append(left, true)
			shardIndex += 1
		} else {
			left = append(left, false)
			shardIndex -= 1
		}
		//如果是等于，说明是奇数个，没有下一个，那么用空的Hash来替代
		if len(l) == int(shardIndex) {
			layer = append(layer, common.Hash{})
		} else {
			layer = append(layer, l[shardIndex])
		}
		shardIndex /= 2 //这个位置是下一层计算得出的index，所以需要取和他一起取hash的另一个值
	}
	return &MerkleTreeProof{
		Proof: layer,
		Left:  left,
	}
}

// shard肯定是一个偶数个数的
func (mt *MerkleTree) MakeMultiProof(shard []bool) *MultiMerkleProof {
	layerProof := make([][]common.Hash, len(mt.Layer))
	layerFlag := make([][]uint8, len(mt.Layer))

	for i, layer := range mt.Layer {
		if len(layer) == 1 {
			break
		}
		layerProof[i] = make([]common.Hash, 0)
		layerFlag[i] = make([]uint8, 0)

		for j := 0; j < len(shard); {
			if shard[j] && shard[j+1] {
				//如果都存在，那么为
				layerFlag[i] = append(layerFlag[i], next)
				shard[j/2] = true
			} else if shard[j] && !shard[j+1] {
				layerFlag[i] = append(layerFlag[i], left)
				if len(layer) == j+1 {
					layerProof[i] = append(layerProof[i], common.Hash{})
				} else {
					layerProof[i] = append(layerProof[i], layer[j+1])
				}
				shard[j/2] = true
			} else if !shard[j] && shard[j+1] {
				layerFlag[i] = append(layerFlag[i], right)
				//奇数个，自己补充一个空hash
				//if len(mt.Layer[i]) == j {
				//	layerProof[i] = append(layerProof[i], common.Hash{})
				//} else {
				layerProof[i] = append(layerProof[i], layer[j])
				//}
				shard[j/2] = true
			} else {
				//如果都没有，那么就不存在下一个
				shard[j/2] = false
			}
			j += 2
		}
		end := len(shard) / 2
		//截断shard，如果是奇数，补充一位到偶数
		if end != 1 && end%2 != 0 {
			shard[end] = false
			shard = shard[:end+1]
		} else {
			shard = shard[:end]
		}

	}
	return &MultiMerkleProof{
		LayerProof: layerProof,
		LayerFlag:  layerFlag,
	}
}

func (mt *MerkleTree) GetRoot() common.Hash {
	return mt.Layer[len(mt.Layer)-1][0]
}

type MerkleTreeProof struct {
	Proof []common.Hash
	Left  []bool
	Nil   bool //rlp编码不可以为空，用一个nil字段，表示这个proof是不是空的。如果是自己的，那么就是false（不是空的，只是没写），如果是别人分片的，那就有可能为true，是空的
}

// 根据自己和当前分片之前的跨分片交易出发，依次和路径取Hash，最终获取Root的Hash
func (mtp *MerkleTreeProof) VerifyProof(crossShardRoot common.Hash) common.Hash {
	//只有一个，直接判断
	if len(mtp.Proof) == 0 {
		return crossShardRoot
	}

	var root common.Hash = crossShardRoot
	//如果root为空，且proof只有一个hash,那么就是最终的Root hash（除非只有两个分片），直接返回proof[0]
	if root == common.EmptyHash && len(mtp.Proof) == 1 {
		return mtp.Proof[0]
	}

	test := make([]byte, 64)

	//0+1 =01 +2 =012 +3 = 0123 ROOT
	for index, h := range mtp.Proof {
		if mtp.Left[index] {
			copy(test[32:], h[:])
			copy(test[:32], root[:])
		} else {
			copy(test[32:], root[:])
			copy(test[:32], h[:])
		}

		root = Sha256(test)
		//重写准备下一次得生成
	}
	return root
}

func (mtp *MerkleTreeProof) Print() {
	if !common.PrintMerkleTree {
		return
	}
	log.Debug("---------Proof-------------")
	if mtp.Proof != nil {
		for _, h := range mtp.Proof {
			log.Debug(fmt.Sprintf("%v  ", h.String()))
		}
	}
	log.Debug("----------end--------------")
}

type MultiMerkleProof struct {
	LayerProof [][]common.Hash
	LayerFlag  [][]uint8
}

const (
	next uint8 = iota
	left
	right
)

func (mmp *MultiMerkleProof) VerifyProof(proofs []common.Hash) common.Hash {
	if len(proofs) == 0 {
		proofs = []common.Hash{common.EmptyHash}
	}
	proofOffset := 0
	layerIndexOffset := 0
	layerProofOffser := 0
	// 我们使用一个标志位，存储每层的Proofs列表中的Hash值怎么用，并且把其再存储如proofs中，减少内存申请
	test := make([]byte, 64)
	//进行下一层的遍历
	for i, flags := range mmp.LayerFlag {
		layerIndexOffset = 0
		layerProofOffser = 0
		proofOffset = 0
		//遍历本层的所有flag
		for len(flags) > layerIndexOffset {
			//根据标志位，生成下一层的proof
			switch flags[layerIndexOffset] {
			case next:
				copy(test[:32], proofs[proofOffset][:])
				copy(test[32:], proofs[proofOffset+1][:])
				proofOffset += 2
				proofs[layerIndexOffset] = Sha256(test)
			case left:
				copy(test[:32], proofs[proofOffset][:])
				copy(test[32:], mmp.LayerProof[i][layerProofOffser][:])
				proofOffset += 1
				layerProofOffser++
				proofs[layerIndexOffset] = Sha256(test)
			case right:
				copy(test[:32], mmp.LayerProof[i][layerProofOffser][:])
				if len(proofs) == proofOffset {
					fmt.Println("err")
				}
				copy(test[32:], proofs[proofOffset][:])
				proofOffset += 1
				layerProofOffser++
				proofs[layerIndexOffset] = Sha256(test)
			}
			//增加标志位的偏移
			layerIndexOffset++
		}
	}
	return proofs[0]
}
