package core

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"testing"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/log"
)

//4.增加分片方案后，不同分片下对带宽的影响
// 柱状图加折线图，不同分片下编码数据的大小，横坐标第一层是分片数量，第二层是不同高度下的交易，纵坐标是原数据量大小和增加后的数据大小（重叠），
// 折线图是跨分片交易和分片交易的比例（类似概率密度）
func TestShard(t *testing.T) {
	log.Create("", "testShard", 4)
	log.Init(4, 15)
	ws := NewWorldState()
	bc := block.NewBlock(1, common.Hash{}, common.Uint64ToByte(0), common.EmptyAddress)

	f := excelize.NewFile()
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	for bn := 0; bn < 17; bn++ {
		ns := bn * 1000000
		// Create a new sheet.
		sheet := fmt.Sprintf("%v", ns)
		index, err := f.NewSheet(sheet)
		if err != nil {
			fmt.Println(err)
			return
		}
		idx := 2
		col := 2

		//获取交易
		alltxs := GetShardTxs(ns) //makeTx()

		//size := 6
		size := 50000
		txs := alltxs[:]
		log.Info("txs", "size", len(txs))

		setvalue(1, 3, "区块大小", sheet, f)
		setvalue(1, 4, "总大小", sheet, f)
		setvalue(1, 5, "分片大小", sheet, f)

		for i := 0; i < len(txs); i += size {

			setvalue(col, 2, "数据大小", sheet, f)
			setvalue(col+1, 2, "分片交易量", sheet, f)
			setvalue(col+2, 2, "跨分片交易", sheet, f)

			setvalue(col, 1, i, sheet, f)
			setvalue(col+1, 1, i+size, sheet, f)
			log.Info(fmt.Sprintf("*******-----------------%v----------------------", i))
			var stxs []*block.Transaction
			if i+size < len(txs) {
				stxs = txs[i : i+size]

			} else {
				stxs = txs[i:]
			}
			sbl, root, err := ws.ExecuteBc(bc, stxs)
			bc.SetLedgerHash(root)
			bcdata, err := rlp.EncodeToBytes(bc)
			if err != nil {
				t.Log(err)
				return
			}

			bc.ShardBody = sbl
			if err != nil {
				t.Log(err)
				return
			}

			data, err := rlp.EncodeToBytes(bc)
			if err != nil {
				t.Log(err)
				return
			}

			bldata, err := rlp.EncodeToBytes(sbl)
			if err != nil {
				t.Log(err)
				return
			}

			bccell, err := excelize.CoordinatesToCellName(col, idx+1)
			if err != nil {
				t.Log(err)
			}
			f.SetCellValue(sheet, bccell, len(bcdata))

			allcell, err := excelize.CoordinatesToCellName(col, idx+2)
			if err != nil {
				t.Log(err)
			}
			f.SetCellValue(sheet, allcell, len(data))

			sbcell, err := excelize.CoordinatesToCellName(col, idx+3)
			if err != nil {
				t.Log(err)
			}
			f.SetCellValue(sheet, sbcell, len(bldata))

			for shardIndex, sb := range sbl {
				cell1, err := excelize.CoordinatesToCellName(col, idx+4+shardIndex)
				if err != nil {
					t.Log(err)
				}
				sharddata, err := rlp.EncodeToBytes(sb)
				if err != nil {
					t.Log(err)
					return
				}
				f.SetCellValue(sheet, cell1, len(sharddata))

				cell2, err := excelize.CoordinatesToCellName(col+1, idx+4+shardIndex)
				if err != nil {
					t.Log(err)
				}
				f.SetCellValue(sheet, cell2, len(sb.Txs))

				setvalue(1, idx+4+shardIndex, shardIndex, sheet, f)
			}

			crossShardCount := make([]int, common.InitDirectShardCount)
			localCount := make([]int, common.InitDirectShardCount)
			for si, sb := range sbl {
				crossShardCount[si] = 0
				localCount[si] = 0
				for _, tx := range sb.Txs {
					if common.Shard(tx.From) != common.Shard(common.BytesToAddress(tx.Data[8:28])) {
						crossShardCount[si]++
					}
				}
				setvalue(col+2, idx+4+si, crossShardCount[si], sheet, f)
			}
			col += 3
		}

		f.SetActiveSheet(index)
	}
	// Save spreadsheet by the given path.
	if err := f.SaveAs(fmt.Sprintf("ShardExprimentData_%v.xlsx", common.InitDirectShardCount)); err != nil {
		fmt.Println(err)
	}
}

func setvalue(col, idx int, value interface{}, sheet string, f *excelize.File) error {
	cell, err := excelize.CoordinatesToCellName(col, idx)
	if err != nil {
		return err
	}
	return f.SetCellValue(sheet, cell, value)
}

func GetShardTxs(index int) []*block.Transaction {
	stxs := readFile(index)
	txs := make([]*block.Transaction, len(stxs))
	for i, tx := range stxs {
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
	}
	return txs
}
