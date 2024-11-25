package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
	"io/ioutil"
	"math/big"
	"sync"
	"testing"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/hexutil"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/crypto/schnorr"
	"witCon/log"
)

// todo 画出图来
// 交易生成签名，交易提前签名
// schnorr签名部分 - 地址映射 - 聚合签名 - 批量验证
// 通信部分 - 建立验证节点连接，发送验证节点
// 节点部分 - 建立验证节点认知
func TestHashes(t *testing.T) {
	mt := &crypto.MerkleTree{}
	hashes := []common.Hash{
		common.HexToHash("0xf7b50260c0524d1e86bb97885e54ab630227331e40f3b90f957cadd8cc8b4cab"),

		common.HexToHash("0x9cf3210dad819e6c6636e2d161895c20c557428314fedd37cf4ed100e8e22e4a"),

		common.HexToHash("0xb5523b8b2bc870a53deebcfd047c60f75de5c074f0200cad36976b98f356c883"),

		common.HexToHash("0x6336c95be1f946ba3e788f41c0d90eaced8e6136ed86b3ce1e67c2ada40b1f66"),
	}
	mt.MakeTree(hashes)
	t.Log(mt.GetRoot().String())

	bytes := [][]byte{[]byte{}, []byte{0x01}}
	proofs := make([]*crypto.MerkleTreeProof, 2)
	proofs[0] = nil
	proofs[1] = &crypto.MerkleTreeProof{Proof: []common.Hash{common.EmptyHash, common.EmptyHash}, Left: []bool{true}}
	type tt struct {
		Bytes  [][]byte
		Proofs []*crypto.MerkleTreeProof
	}
	t1 := &block.ShardBody{
		ReadState:  bytes,
		ShardProof: proofs,
	}
	bcdata, err := rlp.EncodeToBytes(t1)
	if err != nil {
		t.Error(err)
		return
	}
	t2 := &block.ShardBody{}
	err = rlp.DecodeBytes(bcdata, &t2)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTimeLoop(t *testing.T) {
	timer := time.NewTimer(0 * time.Second)
	go func() {
		<-timer.C
		for {
			select {
			case <-timer.C:
				t.Log("timerLoop occurTimeOut")
			}
		}
	}()
	for {
		timer.Reset(1000 * time.Millisecond)
		time.Sleep(1500 * time.Millisecond)
	}
}

func TestExecuteTx(t *testing.T) {
	log.Create("", "testExecute", 4)
	log.Init(4, 15)
	ws := NewWorldState()
	ws2 := NewWorldState()
	bc := block.NewBlock(1, common.Hash{}, common.Uint64ToByte(0), common.EmptyAddress)
	//打开Excel文件开始记录
	//f, err := excelize.OpenFile("ExprimentData.xlsx")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	writeExcel := false
	f := excelize.NewFile()
	sheet := fmt.Sprintf("%v", common.InitDirectShardCount)
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	// Create a new sheet.
	index, err := f.NewSheet(sheet)
	if err != nil {
		fmt.Println(err)
		return
	}

	idx := 3
	col := 2

	//获取交易
	alltxs := GetTxs() //makeTx()

	//size := 6
	size := 6000
	txs := alltxs[:]
	log.Info("txs", "size", len(txs))

	t0 := time.Now().UnixMilli()
	for i := 0; i < len(txs); i += size {
		log.Info(fmt.Sprintf("*******-----------------%v----------------------", i))
		var stxs []*block.Transaction
		if i+size < len(txs) {
			stxs = txs[i : i+size]

		} else {
			stxs = txs[i:]
		}
		t1 := time.Now().UnixMilli()
		sbl, root, err := ws.ExecuteBc(bc, stxs)
		t2 := time.Now().UnixMilli()
		bc.SetLedgerHash(root)
		bc.ShardBody = sbl
		if err != nil {
			t.Log(err)
			return
		}

		_, _, err = ws.ExecuteBc(bc, stxs)
		if err != nil {
			t.Log(err)
			return
		}

		bcdata, err := rlp.EncodeToBytes(bc)
		if err != nil {
			t.Log(err)
			return
		}
		//sbl2 := &block.ShardBody{}
		err = rlp.DecodeBytes(bcdata, bc)
		if err != nil {
			t.Log(err)
			return
		}

		data, err := rlp.EncodeToBytes(stxs)
		if err != nil {
			t.Log(err)
			return
		}
		data2, err := rlp.EncodeToBytes(sbl)
		if err != nil {
			t.Log(err)
			return
		}
		rate := len(data2) * 100 / len(data)
		log.Info("data size", "res", len(data), "des", len(data2), "rate", rate, "roor", root.String())

		t3 := time.Now().UnixMilli()
		verifyPool, _ := ants.NewPool(common.InitDirectShardCount * 2)

		t10 := time.Now().UnixMilli()
		err = ws2.VerifyShardMulti(bc, sbl, verifyPool)
		t11 := time.Now().UnixMilli()
		t.Log("multi", "time", t2-t1, "mtime", t11-t10)
		//for index, sb := range sbl {
		//	err := ws2.VerifyShard(bc, sb, uint16(index))
		if err != nil {
			//t.Log(size)
			t.Log(t2 - t1)
			t.Log(err)
			return
		}
		//}
		t4 := time.Now().UnixMilli()
		log.Info("time", "pack", t2-t1, "verify", t4-t3)
		log.Info("---------------------------------")
		//todo 需要补充一下各分片的大小的列表
		//变量是，分片数、交易数

		if writeExcel {
			cell1, err := excelize.CoordinatesToCellName(col, idx)
			if err != nil {
				t.Log(err)
			}
			cell2, err := excelize.CoordinatesToCellName(col+1, idx)
			if err != nil {
				t.Log(err)
			}
			f.SetCellValue(sheet, cell1, len(data))
			f.SetCellValue(sheet, cell2, size)
			for _, sb := range sbl {
				cell1, err := excelize.CoordinatesToCellName(col, idx+1)
				if err != nil {
					t.Log(err)
				}
				cell2, err := excelize.CoordinatesToCellName(col+1, idx+1)
				if err != nil {
					t.Log(err)
				}
				data3, err := rlp.EncodeToBytes(sb)
				if err != nil {
					t.Log(err)
					return
				}
				f.SetCellValue(sheet, cell1, len(data3))
				f.SetCellValue(sheet, cell2, len(sb.Txs))
				idx++
			}
			col += 3
			idx = 3
		}
		//各分片的交易数量，各分片的大小
		//
		//还可以是箱线图，对于不同高度下，最多的分片和最少的分片，中位数，四分位数
	}

	tend := time.Now().UnixMilli()
	log.Error("All time", "count", tend-t0)
	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// Save spreadsheet by the given path.
	if err := f.SaveAs("ExprimentData.xlsx"); err != nil {
		fmt.Println(err)
	}
}

func TestProof(t *testing.T) {
	mt := crypto.MerkleTreeProof{
		Proof: []common.Hash{common.HexToHash("0x9de232cacd9b2f097ed564ac189413ee6025587929dac16580f61b48b0971658"), common.HexToHash("0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b")},
		Left:  []bool{true, true},
	}
	mt.VerifyProof(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))

	//t.Log(root.String())

	h := common.Hash{}
	two := make([]byte, 64)
	h1 := common.HexToHash("0x9de232cacd9b2f097ed564ac189413ee6025587929dac16580f61b48b0971658")
	t.Log(h1.String())
	h2 := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	t.Log(h2.String())
	copy(two[32:], h1[:])
	copy(two[:32], h2[:])
	t.Log(hexutil.Encode(two))
	h = crypto.Sha256(two)
	t.Log(h.String())
	//two := make([]byte, 64)
	//copy(two[0:32], common.Hex2Bytes("0x0000000000000000000000000000000000000000000000000000000000000000")[:])
	//copy(two[32:], common.Hex2Bytes("0x9de232cacd9b2f097ed564ac189413ee6025587929dac16580f61b48b0971658")[:])
	//t.Log(h.String())
}

// todo 需要补充一下以太坊的数据集
func makeTx() []*block.Transaction {
	size := 200000
	tx := make([]*block.Transaction, size)
	for i := 0; i < size; i++ {
		tx[i] = block.NewTx(common.EmptyAddress, common.EmptyAddress, big.NewInt(1000), 1)
	}
	return tx
}

func ReadTxs() []*block.Transaction {
	stxs := readFile(0)
	txs := make([]*block.Transaction, len(stxs))
	for i, tx := range stxs {
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
	}
	return txs
}

func TestTxs(t *testing.T) {
	count := 0
	for i := 0; i <= 10000000; i += 1000000 {
		txs := readFile(i)
		count += len(txs)
	}
	t.Log(count)
}

func readFile(index int) []*sTxs {
	txpath := fmt.Sprintf("%s\\%v", "E:\\ethdata\\", index)
	b, err := ioutil.ReadFile(txpath)
	if err != nil {
		log.Error("文件decode读取失败", "err", err)
		return nil
	}
	txs := make([]*sTxs, 0)
	err = rlp.DecodeBytes(b, &(txs))
	if err != nil {
		log.Error("文件decode失败", "err", err)
		return nil
	}
	return txs
}

type sTxs struct {
	From   common.Address
	To     common.Address
	Number uint64
	Amount *big.Int
}

func TestVerifyTxs(t *testing.T) {
	stxs := readETHFile(0)
	stxs = stxs[:200000]
	txs := make([]*block.Transaction, len(stxs))
	for i, tx := range stxs {
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
		txs[i].SetSig(tx.PublicKey, tx.Signature)
	}
	batchLen := len(txs)
	pks := make([][33]byte, batchLen)
	msgs := make([][32]byte, batchLen)
	sigs := make([][64]byte, batchLen)
	//*********************************************
	t1 := time.Now().UnixMilli()
	//
	for i, tx := range txs {
		data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
		if err != nil {
			t.Log("encode err", err.Error())
			return
		}
		hash := crypto.HashSum(data)
		msgs[i] = hash
	}

	t11 := time.Now().UnixMilli()
	for i, tx := range txs {
		pks[i] = crypto.SchnorrPk(tx.Sig[:33])
		sigs[i] = crypto.SchnorrSig(tx.Sig[33:])
	}
	//*********************************************
	t2 := time.Now().UnixMilli()

	succ, err := crypto.VerifyBatchSchnorr(pks, msgs, sigs)
	//*********************************************
	t3 := time.Now().UnixMilli()

	for i, _ := range txs {
		schnorr.Verify(pks[i], msgs[i], sigs[i])
	}
	//*********************************************
	t4 := time.Now().UnixMilli()
	t.Log("encode", t2-t1, "encodeHash", t11-t1, "batch verify", t3-t2, "serial verify", t4-t3)

	if !succ {
		t.Log("verify fail", "err", err)
	} else {
		t.Log("succ")
	}
}

func TestShardVerifyTxs(t *testing.T) {
	stxs := readETHFile(0)
	stxs = stxs[:400000]
	txs := make([]*block.Transaction, len(stxs))
	for i, tx := range stxs {
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
		txs[i].SetSig(tx.PublicKey, tx.Signature)
	}
	t4 := time.Now().UnixMilli()

	verifySigPool, _ := ants.NewPool(24)

	shardSize := 100
	shardlen := len(txs) / shardSize
	var wg sync.WaitGroup
	for i := 0; i < shardlen; i++ {
		wg.Add(1)
		start := i * shardSize
		end := (i + 1) * shardSize
		verifySigPool.Submit(func() {
			for j := start; j < end; j++ {
				tx := txs[j]
				data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
				if err != nil {
					t.Log("encode err", err.Error())
					return
				}
				hash := crypto.HashSum(data)
				pk := crypto.SchnorrPk(tx.Sig[:33])
				sig := crypto.SchnorrSig(tx.Sig[33:])
				schnorr.Verify(pk, hash, sig)
			}
			wg.Done()
		})
	}
	wg.Wait()

	t5 := time.Now().UnixMilli()
	t.Log("shard verify", t5-t4)
}

func TestShardVerifyECDSATxs(t *testing.T) {
	stxs := readETHECDSAFile(0)
	//stxs = stxs[:100]
	txs := make([]*block.Transaction, len(stxs))
	for i, tx := range stxs {
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
		//t.Log(hex.EncodeToString(tx.Signature))
		txs[i].SetSig(tx.PublicKey, tx.Signature)
	}

	verifySigPool, _ := ants.NewPool(3)

	t4 := time.Now().UnixMilli()
	shardSize := 10
	shardlen := len(txs) / shardSize
	var wg sync.WaitGroup
	for i := 0; i < shardlen; i++ {
		wg.Add(1)
		start := i * shardSize
		end := (i + 1) * shardSize
		verifySigPool.Submit(func() {
			defer wg.Done()
			for j := start; j < end; j++ {
				tx := txs[j]
				data, err := rlp.EncodeToBytes([]interface{}{tx.From, tx.Number, tx.Data})
				if err != nil {
					t.Log("encode err", err.Error())
					return
				}
				hash := crypto.HashSum(data)
				pk, err := crypto.SigToPub(hash[:], tx.Sig[33:])
				pks := crypto.CompressPubKey(pk)
				if err != nil || !bytes.Equal(pks, tx.Sig[:33]) {
					t.Error("pk fail", "err", err, "pk", hex.EncodeToString(pks), "tx", hex.EncodeToString(tx.Sig[:33]))
					return
				}
			}
		})
	}
	wg.Wait()

	t5 := time.Now().UnixMilli()

	t.Log("shard verify size", len(stxs))
	t.Log("shard verify ecdsa", t5-t4)
}
