package core

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/log"
	"witCon/stat"
)

type TxGenerator struct {
	TxPool
	txs            []*block.Transaction
	addr           common.Address
	sk             *ecdsa.PrivateKey
	ws             *WorldState
	exchangeGoPool *ants.Pool
	nonce          uint64
}

func NewTxGenerator(sk *ecdsa.PrivateKey, readTx bool) *TxGenerator {
	tg := &TxGenerator{
		ws:    NewWorldState(),
		nonce: 0,
	}
	tg.TxPool = *NewTxPool()
	tg.exchangeGoPool, _ = ants.NewPool(3)
	tg.addr = crypto.PubKeyToAddress(sk.PublicKey)
	tg.sk = sk
	//tg.readFile()

	if readTx {
		tg.txs = GetTxs()
	}
	return tg
}

// 预先生成一批交易
func (tg *TxGenerator) GenerateTx(num int, time int) {
	log.Info("GenerateTx", "num", num, "time", time)
	if tg.txs == nil || len(tg.txs) == 0 {
		log.Info("GenerateTx2", "num", num, "time", time)
		size := num * time
		log.Info("GenerateTx2", "size", size)
		timeTxs := make([]*block.Transaction, size)
		for i := 0; i < size; i++ {
			tx := block.NewTx(tg.addr, tg.addr, big.NewInt(10), tg.nonce)
			tg.nonce++
			tg.Sign(tx)
			timeTxs[i] = tx
			if i%1000 == 0 {
				log.Info("GenerateTx4", "i", i)
			}
		}
		tg.txs = timeTxs
		tg.writeFile(timeTxs)
		log.Info("GenerateTx3", "txs", len(tg.txs))
	}
}

// 开始以一定速率往交易池中推送交易
func (tg *TxGenerator) StartTxRate(num int) {
	timer := time.NewTimer(0)
	t := 0
	num = num / 10
	addTxCh := make(chan []*block.Transaction, 1000)
	log.NewGoroutine(func() {
		for {
			select {
			case txs := <-addTxCh:
				tg.exchangeGoPool.Submit(func() {
					tg.TxPool.AddTx(txs)
				})
			}
		}
	})
	for {
		select {
		case <-timer.C:
			//todo 发送
			start := t * num
			end := (t + 1) * num
			if end > len(tg.txs) {
				return
			}
			//tg.TxPool.AddTx(tg.txs[start:end])
			txs := tg.txs[start : end+1]
			stat.Instance.DoLock(func() {
				for _, tx := range txs {
					stat.Instance.OnTxIn(tx.TxHash)
				}
			})
			addTxCh <- txs
			timer.Reset(100 * time.Millisecond)
			t++
		}
	}
}

func (tg *TxGenerator) Sign(tx *block.Transaction) {
	if tx.From != tg.addr {
		log.Error("sign with fail addr")
		return
	}
	sig, _ := crypto.Sign(tx.TxHash.Bytes(), tg.sk)
	tx.Sign(sig)
}

func (tg *TxGenerator) readFile() {
	txpath := filepath.Join(common.DataPath, common.DefaultTxPath) //fmt.Sprintf("%s\\%v", common.DataPath, common.DefaultTxPath)
	b, err := ioutil.ReadFile(txpath)
	if err != nil {
		log.Error("文件decode读取失败", "err", err)
		return
	}
	tg.txs = make([]*block.Transaction, 0)
	err = rlp.DecodeBytes(b, &(tg.txs))
	if err != nil {
		log.Error("文件decode失败", "err", err)
		return
	}
	return
}

func (tg *TxGenerator) writeFile(txs []*block.Transaction) {
	txpath := filepath.Join(common.DataPath, common.DefaultTxPath)
	fmt.Println("create tx", "txpath", txpath)
	if _, err := os.Stat(txpath); os.IsNotExist(err) {
		b, err := rlp.EncodeToBytes(txs)

		err = ioutil.WriteFile(txpath, b, 0777)
		if err != nil {
			fmt.Println("文件写入失败", err)
			return
		}
		fmt.Println("文件写入成功", err)
		return
	}
}

func GetTxs() []*block.Transaction {
	stxs := readETHECDSAFile(0)
	txs := make([]*block.Transaction, len(stxs))
	for i, tx := range stxs {
		// txs[i] = block.NewTx(tx.From, stxs[0].To, tx.Amount, tx.Number)
		txs[i] = block.NewTx(tx.From, tx.To, tx.Amount, tx.Number)
		txs[i].SetSig(tx.PublicKey, tx.Signature)
	}
	return txs
}

func readETHFile(index int) []*SampleTxs {
	txpath := filepath.Join("E:\\ethdata\\sig\\", strconv.Itoa(index))
	b, err := ioutil.ReadFile(txpath)
	if err != nil {
		log.Error("文件decode读取失败", "err", err)
		return nil
	}
	txs := make([]*SampleTxs, 0)
	err = rlp.DecodeBytes(b, &(txs))
	if err != nil {
		log.Error("文件decode失败", "err", err)
		return nil
	}

	return txs
}

func readETHECDSASliceFile(index int, startLen int) []*SampleTxs {
	txpath := fmt.Sprintf("%s\\%v\\slice_%v", common.EthTxPath, index, startLen)
	// 打开文件
	b, err := ioutil.ReadFile(txpath)
	if err != nil {
		log.Error("文件decode读取失败", "err", err)
		return nil
	}
	txs := make([]*SampleTxs, 0)
	err = rlp.DecodeBytes(b, &(txs))
	if err != nil {
		log.Error("文件decode失败", "err", err)
		return nil
	}

	return txs
}

func readETHECDSAFileLen(index int, len int) []*SampleTxs {
	txpath := filepath.Join(common.EthTxPath, strconv.Itoa(index))
	b, err := ioutil.ReadFile(txpath)
	if err != nil {
		log.Error("文件decode读取失败", "err", err)
		return nil
	}
	txs := make([]*SampleTxs, 0)
	err = rlp.DecodeBytes(b, &(txs))
	if err != nil {
		log.Error("文件decode失败", "err", err)
		return nil
	}

	return txs[:len]
}

func readETHECDSAFile(index int) []*SampleTxs {
	txpath := filepath.Join(common.EthTxPath, strconv.Itoa(index))
	b, err := ioutil.ReadFile(txpath)
	if err != nil {
		log.Error("文件decode读取失败", "err", err)
		return nil
	}
	txs := make([]*SampleTxs, 0)
	err = rlp.DecodeBytes(b, &(txs))
	if err != nil {
		log.Error("文件decode失败", "err", err)
		return nil
	}

	return txs
}

type SampleTxs struct {
	From      common.Address
	To        common.Address
	Number    uint64
	Amount    *big.Int
	PublicKey [33]byte
	Signature []byte
}
