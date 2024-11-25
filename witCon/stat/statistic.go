package stat

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"sync"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/log"
)

var (
	Instance = &Center{}
)

// 统计的数据
// ok 消息输出量
// ok 消息输入量
// ok TPS,设置每个交易的大小，根据一个区块中可以包含多少笔判断
// ok 每秒的区块确认数
// ok 区块的确认时间
// ok 每个区块的通信大小 ：统计每个区块Hash下的通信量和区块大小的对比（有意义嘛？）
//
// 在网络抖动情况下，上述的值（按理也不会有什么变化）
type Center struct {
	sendCount     uint64
	lastSendCount uint64
	recCount      uint64
	lastRecCount  uint64

	pending   sync.Map
	confirmed map[common.Hash]struct{}

	pendingTx map[common.Hash]int64

	timesCFT        int64
	accurateTimeCFT int64
	avgTimeCFT      int64

	times        []int64
	accurateTime int64
	avgTime      int64

	txCount        int64
	lastTxCount    int64
	accurateTxTime int64
	avgTxTime      int64

	txCountCFT        int64
	timesTxCFT        int64
	accurateTxTimeCFT int64
	avgTxTimeCFT      int64

	blockCount     int64
	lastBlockCount int64

	Size int64

	lock sync.RWMutex

	//Excel 读写相关
	nextCol    int
	nextIdx    int
	exlFile    *excelize.File
	sheetIndex int
	sheetName  string
	si         *SystemInfo
	time_count int
	filename   string
}

func (c *Center) Init(cfg *common.Config) {
	c.pendingTx = make(map[common.Hash]int64)
	c.confirmed = make(map[common.Hash]struct{})
	c.times = make([]int64, 0)
	c.Size = 1
	datapath := fmt.Sprintf("%s\\", common.DataPath)
	c.si = &SystemInfo{Path: datapath}
	c.si.mainLoop()
	c.InitExcel(cfg)
	log.NewGoroutine(c.Statistic)
}

var statisticDuration = 1 * time.Second

func (c *Center) Statistic() {
	t := time.NewTimer(statisticDuration)
	for {
		select {
		case <-t.C:
			send, rec := c.GetNetRwSize()
			blockAdd := c.blockCount - c.lastBlockCount
			txAdd := c.txCount - c.lastTxCount
			bps := 0
			if blockAdd == 0 {
				bps = 0
				blockAdd = 1
			} else {
				bps = int(blockAdd) / 1
			}
			c.lastBlockCount = c.blockCount
			c.lastTxCount = c.txCount
			if c.blockCount > 0 {
				log.Info("Statistic", "rtt", common.Rtt, "net_send", send/1024/1024/1, "net_rec", rec/1024/1024/1,
					"stat_len", len(c.times), "cft_time", c.avgTimeCFT, "fin_time", c.avgTime,
					"block_count", c.blockCount, "bps", bps, "size_rate", (int64(send+rec)*100)/(blockAdd*c.Size),
					"tx_count", c.txCount, "tx_time", c.avgTxTime, "txPS", txAdd, "tx_cft_time", c.avgTxTimeCFT)
				c.WriteTPSExcel(int(txAdd), int(c.avgTxTime), bps, c.si.AllCpuTotalPercent, c.si.NetBytesRecvSpeed, c.si.NetBytesSentSpeed)
			}
			t.Reset(statisticDuration)
		}
	}
}

func (c *Center) OnFdWrite(size int) {
	c.sendCount += uint64(size)
}

func (c *Center) OnFdRead(size int) {
	c.recCount += uint64(size)
}

func (c *Center) GetNetRwSize() (send uint64, rec uint64) {
	ts := c.sendCount
	tr := c.recCount
	send = ts - c.lastSendCount
	rec = tr - c.lastRecCount
	c.lastSendCount = ts
	c.lastRecCount = tr
	return
}

func (c *Center) OnProtoIn() {

}

func (c *Center) OnProtoOut() {

}

func (c *Center) OnBlockIn(hash common.Hash) {
	if _, ok := c.pending.Load(hash); !ok {
		c.pending.Store(hash, time.Now().UnixMicro())
	}
}

func (c *Center) OnBlockConfirm(bc *block.Block) {
	hash := bc.Hash
	if _, ok := c.confirmed[hash]; ok {
		return
	}
	c.confirmed[hash] = struct{}{}
	if startTime, ok := c.pending.Load(hash); ok {
		t := time.Now().UnixMicro() - startTime.(int64)
		c.timesCFT++
		c.accurateTimeCFT += t
		c.avgTimeCFT = c.accurateTimeCFT / c.timesCFT
	}

	if bc.Txs != nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		for _, tx := range bc.Txs {
			txHash := tx.TxHash
			if startTime, ok := c.pendingTx[txHash]; ok {
				c.txCountCFT++
				t := time.Now().UnixMicro() - startTime
				c.accurateTxTimeCFT += t
				c.avgTxTimeCFT = c.accurateTxTimeCFT / c.txCountCFT
			}
		}
	}

}

func (c *Center) OnBlockWrite(hash common.Hash, size int) {
	if startTime, ok := c.pending.Load(hash); ok {
		t := time.Now().UnixMicro() - startTime.(int64)
		c.times = append(c.times, t)
		c.accurateTime += t
		c.avgTime = c.accurateTime / int64(len(c.times))
	}
	if c.Size == 1 {
		c.Size = int64(size)
	}
	c.blockCount++
}

func (c *Center) DoLock(task func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	task()
}

func (c *Center) OnTxIn(hash common.Hash) {
	c.pendingTx[hash] = time.Now().UnixMicro()
}

func (c *Center) OnCTxWrite(size int) {
	c.txCount += int64(size)
}

func (c *Center) OnTxWrite(hash common.Hash) {
	if startTime, ok := c.pendingTx[hash]; ok {
		c.txCount++
		t := time.Now().UnixMicro() - startTime
		c.accurateTxTime += t
		c.avgTxTime = c.accurateTxTime / c.txCount
	}
	delete(c.pendingTx, hash)
}

func (c *Center) Print() {
	send, rec := c.GetNetRwSize()
	blockAdd := c.blockCount - c.lastBlockCount
	txAdd := c.txCount - c.lastTxCount
	if blockAdd == 0 {
		blockAdd = 1
	}
	c.lastBlockCount = c.blockCount
	c.lastTxCount = c.txCount
	log.Info("End-Statistic", "net_send", send/1024/1024/1, "net_rec", rec/1024/1024/1,
		"stat_len", len(c.times), "cft_time", c.avgTimeCFT, "fin_time", c.avgTime,
		"block_count", c.blockCount, "bps", blockAdd/1, "size_rate", (int64(send+rec)*100)/(blockAdd*c.Size),
		"tx_count", c.txCount, "tx_time", c.avgTxTime, "txPS", txAdd, "tx_cft_time", c.avgTxTimeCFT)
}

func GetExlFile(cfg *common.Config) string {
	return fmt.Sprintf("send_%v_node_%v_pre_%v_vs_%v_shard_%v_signCore_%v_shardCore_%v_%v_%v.xlsx",
		cfg.TxSize, len(cfg.NodeList), cfg.PrePack, cfg.Schnorr, cfg.ShardCount, cfg.SignVerifyCore, cfg.ShardVerifyCore, cfg.VerifyNode, cfg.Name.String())

}

func (c *Center) InitExcel(cfg *common.Config) {

	c.filename = GetExlFile(cfg)
	c.exlFile = excelize.NewFile()
	sheet := fmt.Sprintf("%v", common.InitDirectShardCount)
	//defer func() {
	//	// Close the spreadsheet.
	//	if err := c.exlFile.Close(); err != nil {
	//		fmt.Println(err)
	//	}
	//}()
	// Create a new sheet.
	index, err := c.exlFile.NewSheet(sheet)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.sheetIndex = index
	c.sheetName = sheet

	cell, err := excelize.CoordinatesToCellName(1, 1)
	c.exlFile.SetCellValue(c.sheetName, cell, "性能/消耗")

	cell, err = excelize.CoordinatesToCellName(1, 3)
	c.exlFile.SetCellValue(c.sheetName, cell, "TPS")
	cell, err = excelize.CoordinatesToCellName(1, 4)
	c.exlFile.SetCellValue(c.sheetName, cell, "Latency")
	cell, err = excelize.CoordinatesToCellName(1, 5)
	c.exlFile.SetCellValue(c.sheetName, cell, "BPS")
	cell, err = excelize.CoordinatesToCellName(1, 6)
	c.exlFile.SetCellValue(c.sheetName, cell, "CPU_Used")
	cell, err = excelize.CoordinatesToCellName(1, 7)
	c.exlFile.SetCellValue(c.sheetName, cell, "Net_in")
	cell, err = excelize.CoordinatesToCellName(1, 8)
	c.exlFile.SetCellValue(c.sheetName, cell, "Net_out")

	// Set active sheet of the workbook.
	c.exlFile.SetActiveSheet(c.sheetIndex)
	// Save spreadsheet by the given path.
	if err := c.exlFile.SaveAs(c.filename); err != nil {
		fmt.Println(err)
	}

	c.nextIdx = 3
	c.nextCol = 2
}

func (c *Center) WriteTPSExcel(tps int, latency int, bps int, cpuUsed float64, in uint64, out uint64) {

	r1, r2 := DBlockTimeTrace.StatTB()
	log.Error(r1)
	log.Error(r2)
	log.Error("excel", "tps", tps, "latency", latency, "cpuUsed", cpuUsed)

	cell, err := excelize.CoordinatesToCellName(c.nextCol, 2)
	c.exlFile.SetCellValue(c.sheetName, cell, c.time_count)
	c.time_count++

	cell1, err := excelize.CoordinatesToCellName(c.nextCol, c.nextIdx)
	if err != nil {
		log.Error("get cell tps fail", "err", err)
	}
	c.exlFile.SetCellValue(c.sheetName, cell1, tps)

	cell2, err := excelize.CoordinatesToCellName(c.nextCol, c.nextIdx+1)
	if err != nil {
		log.Error("get cell latency fail", "err", err)
	}
	c.exlFile.SetCellValue(c.sheetName, cell2, latency)

	cell3, err := excelize.CoordinatesToCellName(c.nextCol, c.nextIdx+2)
	if err != nil {
		log.Error("get cell bps fail", "err", err)
	}
	c.exlFile.SetCellValue(c.sheetName, cell3, bps)

	cell4, err := excelize.CoordinatesToCellName(c.nextCol, c.nextIdx+3)
	if err != nil {
		log.Error("get cell cpuUsed fail", "err", err)
	}
	c.exlFile.SetCellValue(c.sheetName, cell4, cpuUsed)

	cell5, err := excelize.CoordinatesToCellName(c.nextCol, c.nextIdx+4)
	if err != nil {
		log.Error("get cell in fail", "err", err)
	}
	c.exlFile.SetCellValue(c.sheetName, cell5, in)

	cell6, err := excelize.CoordinatesToCellName(c.nextCol, c.nextIdx+5)
	if err != nil {
		log.Error("get cell out fail", "err", err)
	}
	c.exlFile.SetCellValue(c.sheetName, cell6, out)

	c.nextCol++

	// Set active sheet of the workbook.
	c.exlFile.SetActiveSheet(c.sheetIndex)
	// Save spreadsheet by the given path.

	if err := c.exlFile.SaveAs(c.filename); err != nil {
		fmt.Println(err)
	}
}
