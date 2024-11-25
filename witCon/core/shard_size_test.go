package core

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
	"testing"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/log"
)

// shard proof
func TestShardWithTx(t *testing.T) {
	common.CompressMerkle = true
	common.MultiMerkle = true
	shardWithTx(t, 5000)

	common.CompressMerkle = true
	common.MultiMerkle = true
	shardWithTx(t, 10000)

	common.CompressMerkle = true
	common.MultiMerkle = true
	shardWithTx(t, 20000)

	//common.CompressMerkle = true
	//common.MultiMerkle = false
	//shardWithTx(t, 40000)

	common.CompressMerkle = false
	common.MultiMerkle = false
	shardWithTx(t, 5000)

	common.CompressMerkle = false
	common.MultiMerkle = false
	shardWithTx(t, 10000)

	common.CompressMerkle = false
	common.MultiMerkle = false
	shardWithTx(t, 20000)

}

func shardWithTx(t *testing.T, size int) { //获取交易
	alltxs := GetSliceTxs() //makeTx()
	alltxs = alltxs[:]
	mtctype := 0
	if common.CompressMerkle {
		if common.MultiMerkle {
			mtctype = 2
		} else {
			mtctype = 1
		}
	}
	txsize := size
	for shard := 4; shard <= 256; shard *= 2 {
		common.InitDirectShardCount = shard
		log.Create("", "testExecute", 4)
		log.Init(2, 15)
		ws := NewWorldState()
		ws2 := NewWorldState()
		bc := block.NewBlock(1, common.Hash{}, common.Uint64ToByte(0), common.EmptyAddress)
		//打开Excel文件开始记录
		//f, err := excelize.OpenFile("ExprimentData.xlsx")
		//if err != nil {
		//	fmt.Println(err)
		//	return
		//}
		writeExcel := true
		f := excelize.NewFile()
		sheet := fmt.Sprintf("shard")
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

		//size := 6
		//size := 200000
		txs := alltxs[:]
		log.Info("txs", "size", len(txs))

		t0 := time.Now().UnixMilli()
		const (
			zeroCol = iota
			initCol
			sb_bc_data_size_col
			tx_bc_data_size_col
			sb_data_size_col
			tx_data_size_col
			execute_time_col
			execute_tx_col
			generate_merkle_col
			verify_time_col
		)
		for i := 0; i < len(txs); i += txsize {
			log.Info(fmt.Sprintf("*******-----------------%v----------------------", i))
			var (
				sb_bc_data_size      int
				tx_bc_data_size      int
				sb_data_size         int
				tx_data_size         int
				execute_time         int64
				execute_tx_time      int64
				generate_merkle_time int64
				verify_time          int64
			)
			var stxs []*block.Transaction
			if i+txsize < len(txs) {
				stxs = txs[i : i+txsize]
			} else {
				stxs = txs[i:]
			}
			t1 := time.Now().UnixMicro()
			sbl, root, execute_tx_time, generate_merkle_time, err := ws.ExecuteBcTime(bc, stxs)
			t2 := time.Now().UnixMicro()
			bc.SetLedgerHash(root)
			bc.ShardBody = sbl
			if err != nil {
				t.Log(err)
				return
			}

			bcdata, err := rlp.EncodeToBytes(bc)
			if err != nil {
				t.Log(err)
				return
			}
			sb_bc_data_size = len(bcdata)
			bc.ShardBody = nil
			bc.SetTxs(stxs)
			txdata, err := rlp.EncodeToBytes(bc)
			if err != nil {
				t.Log(err)
				return
			}
			tx_bc_data_size = len(txdata)

			data, err := rlp.EncodeToBytes(stxs)
			if err != nil {
				t.Log(err)
				return
			}
			tx_data_size = len(data)
			data2, err := rlp.EncodeToBytes(sbl)
			if err != nil {
				t.Log(err)
				return
			}
			sb_data_size = len(data2)

			rate := len(data2) * 100 / len(data)
			t.Log("data size", "res", len(data), "des", len(data2), "rate", rate, "root", root.String())

			spd, _ := rlp.EncodeToBytes(sbl[0].ShardProof)
			rsd, _ := rlp.EncodeToBytes(sbl[0].ReadState)
			tsd, _ := rlp.EncodeToBytes(sbl[0].Txs)
			sod, _ := rlp.EncodeToBytes(sbl[0].SelfOp)
			t.Log("sbl size", "ShardProof", len(spd), "ReadState", len(rsd), "SelfOp", len(sod), "Txs", len(tsd))

			t3 := time.Now().UnixMilli()
			verifyPool, _ := ants.NewPool(common.InitDirectShardCount)

			t10 := time.Now().UnixMicro()
			err = ws2.VerifyShardMulti(bc, sbl, verifyPool)
			t11 := time.Now().UnixMicro()
			execute_time = t2 - t1
			verify_time = t11 - t10

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
			var (
				txSize         []int = make([]int, common.InitDirectShardCount)
				crossOtherSize []int = make([]int, common.InitDirectShardCount)
				toMeSize       []int = make([]int, common.InitDirectShardCount)
				sbDataSize     []int = make([]int, common.InitDirectShardCount)
			)
			bc.SetTxs(nil)
			bc.ShardBody = sbl
			for si, sb := range sbl {
				txSize[si] = len(sb.Txs)
				tbc := bc.ShallowCopyShard([]uint{uint(si)})
				data3, err := rlp.EncodeToBytes(tbc)
				if err != nil {
					t.Log(err)
					return
				}
				sbDataSize[si] = len(data3)
				crossOtherSize[si] = len(sb.ReadState)
				toMeSize[si] = len(sb.SelfOp)
			}
			//变量是，分片数、交易数

			if writeExcel {
				idx := i/txsize + 2
				SetCellValue(f, sheet, initCol, idx, i)
				SetCellValue(f, sheet, sb_bc_data_size_col, idx, sb_bc_data_size)
				SetCellValue(f, sheet, tx_bc_data_size_col, idx, tx_bc_data_size)
				SetCellValue(f, sheet, sb_data_size_col, idx, sb_data_size)
				SetCellValue(f, sheet, tx_data_size_col, idx, tx_data_size)
				SetCellValue(f, sheet, execute_time_col, idx, execute_time)
				SetCellValue(f, sheet, execute_tx_col, idx, execute_tx_time)
				SetCellValue(f, sheet, generate_merkle_col, idx, generate_merkle_time)
				SetCellValue(f, sheet, verify_time_col, idx, verify_time)
				for j := 0; j < common.InitDirectShardCount; j++ {
					SetCellValue(f, sheet, verify_time_col+4*j+1, idx, txSize[j])
					SetCellValue(f, sheet, verify_time_col+4*j+2, idx, sbDataSize[j])
					SetCellValue(f, sheet, verify_time_col+4*j+3, idx, toMeSize[j])
					SetCellValue(f, sheet, verify_time_col+4*j+4, idx, crossOtherSize[j])
				}
			}
			//各分片的交易数量，各分片的大小
			//
			//还可以是箱线图，对于不同高度下，最多的分片和最少的分片，中位数，四分位数
		}

		if writeExcel {
			idx := 1
			SetCellValue(f, sheet, initCol, idx, "数量")
			SetCellValue(f, sheet, sb_bc_data_size_col, idx, "分片总大小")
			SetCellValue(f, sheet, tx_bc_data_size_col, idx, "交易总大小")
			SetCellValue(f, sheet, sb_data_size_col, idx, "分片大小")
			SetCellValue(f, sheet, tx_data_size_col, idx, "交易大小")
			SetCellValue(f, sheet, execute_time_col, idx, "执行时间")
			SetCellValue(f, sheet, execute_tx_col, idx, "执行交易")
			SetCellValue(f, sheet, generate_merkle_col, idx, "生成证明")
			SetCellValue(f, sheet, verify_time_col, idx, "验证时间")
			for j := 0; j < common.InitDirectShardCount; j++ {
				SetCellValue(f, sheet, verify_time_col+4*j+1, idx, fmt.Sprintf("%v：交易", j))
				SetCellValue(f, sheet, verify_time_col+4*j+2, idx, fmt.Sprintf("%v：分片大小", j))
				SetCellValue(f, sheet, verify_time_col+4*j+3, idx, fmt.Sprintf("%v：访问自己", j))
				SetCellValue(f, sheet, verify_time_col+4*j+4, idx, fmt.Sprintf("%v：跨别人", j))
			}
		}

		tend := time.Now().UnixMilli()
		log.Error("All time", "count", tend-t0)
		// Set active sheet of the workbook.
		f.SetActiveSheet(index)
		// Save spreadsheet by the given path.

		if err := f.SaveAs(fmt.Sprintf("ShardExprimentData_%v_%v_%v.xlsx", common.InitDirectShardCount, txsize, mtctype)); err != nil {
			fmt.Println(err)
		}
	}
}

func SetCellValue(f *excelize.File, sheet string, col, row int, value interface{}) error {
	cell1, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, cell1, value)
	return nil
}

func GetSliceTxs() []*block.Transaction {
	allLen := 10
	allTxs := make([]*block.Transaction, allLen*1000000)
	for i := 0; i < allLen; i++ {
		stxs := readETHECDSASliceFile(16000000, i)
		fmt.Println(fmt.Sprintf("read %v", i))
		if stxs != nil {
			for j, tx := range stxs {
				// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
				allTxs[i*1000000+j] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
				allTxs[i*1000000+j].SetSig(tx.PublicKey, tx.Signature)
			}
		} else {
			return allTxs
		}
	}
	return allTxs
}

func GetLenTxs() []*block.Transaction {
	lens := 10000000
	allTxs := make([]*block.Transaction, lens)
	stxs := readETHECDSAFileLen(16000000, lens)
	if stxs != nil {
		for j, tx := range stxs {
			// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
			allTxs[j] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
			allTxs[j].SetSig(tx.PublicKey, tx.Signature)
		}
	} else {
		return allTxs
	}
	return allTxs
}
