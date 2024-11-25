package main

import (
	"crypto/ecdsa"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"time"
	"witCon/common"
	"witCon/consensus"
	"witCon/consensus/consensus_service"
	"witCon/consensus/hotstuff"
	"witCon/consensus/jolteon"
	"witCon/consensus/pbft"
	"witCon/consensus/symphony"
	"witCon/core"
	"witCon/crypto"
	rawbd "witCon/db"
	"witCon/log"
	"witCon/p2p"
)

type Node struct {
	config   *common.Config
	srv      *p2p.Server
	wit      *symphony.Wit
	pbft     *pbft.PBFT
	hotstuff *hotstuff.Hotstuff
	jelteon  *jolteon.Jolteon
	tg       *core.TxGenerator
	stop     chan struct{}
}

func NewNode(config *common.Config) *Node {
	n := &Node{
		config: config,
		stop:   make(chan struct{}),
	}
	return n
}

func (n *Node) Start(dataPath string) {
	//addr, _ := common.Base58ToAddress(n.config.Name)
	//saintes := make([]common.Address, len(n.config.SaintList))
	//for index, saint := range n.config.SaintList {
	//	saintes[index], _ = common.Base58ToAddress(saint)
	//}
	common.InitDirectShardCount = int(n.config.ShardCount)
	common.ShardVerifyCore = n.config.ShardVerifyCore

	common.VerifyNode = n.config.VerifyNode
	if n.config.TxPath != "" {
		common.EthTxPath = n.config.TxPath
	}

	common.ShardVerify = n.config.ShardVerify
	common.SignatureVerify = n.config.Schnorr
	common.SignVerifyCore = n.config.SignVerifyCore
	common.PrePacked = n.config.PrePack
	common.Rtt = int64(n.config.Rtt)
	common.DataPath = dataPath
	sk := n.readSk(dataPath)
	seal := core.NewSeal(sk)
	conss := n.config.Consensus
	common.TxAmount = int(n.config.TxAmount)
	common.TxSize = int(n.config.TxSize)
	consensus_service.ViewChangeDuration = time.Duration(n.config.ViewChangeDuration) * time.Millisecond
	nc := core.NewNodeCluster(n.config.Name, n.config.SaintList, false)
	bc := core.NewBlockchain(rawbd.NewMemDB())
	readTx := nc.Turn() == 0 && len(common.VerifyNode) == 0
	n.tg = core.NewTxGenerator(sk, readTx)
	if nc.Turn() == 0 && len(common.VerifyNode) == 0 {
		n.tg.GenerateTx(common.TxSize, 30)
	}
	ws := core.NewWorldState()
	p := consensus.NewProxy(nc, bc, conss, seal, n.tg, ws)
	peerlen := 0
	if len(n.config.VerifyNode) == 0 {
		if n.config.VerifyCount != 0 {
			peerlen = len(n.config.SaintList) + int(n.config.ShardCount)/len(n.config.SaintList)/int(n.config.VerifyCount)
		} else {
			peerlen = len(n.config.SaintList)
		}
	}
	n.srv = p2p.NewServer(n.config.Name, n.config.NodeList, n.config.IP, peerlen, len(n.config.SaintList), p.Action)
	p.Start(n.srv.PS)
	//n.srv.RegisterProto(p2p.BaseSuite, p)
	n.srv.RegisterProto(p2p.ConsensusSuite, p.ProtoHandler)
	log.Info("show saint config", "len", n.config.SaintList)

	//开启网络延迟动态变化
	//n.RandomRtt()

	if peerlen == 1 {
		p.Action()
	}

	//fix go语言command运行必须要加
	if runtime.GOOS == "windows" {
		log.Error("It is windows!!")
		s := make(chan struct{})
		<-s
	}
}

func (n *Node) readSk(datapath string) *ecdsa.PrivateKey {
	log.Info("readSk", "dataPath", datapath)
	skpath := filepath.Join(datapath, common.DefaultSKPath)
	//skpath := fmt.Sprintf("%s\\%v", datapath, common.DefaultSKPath)

	//文件写入缓冲区
	//reader := bufio.NewReader(file)
	//b := make([]byte, reader.Size())
	b, err := ioutil.ReadFile(skpath)
	if err != nil {
		log.Error("文件读取失败", "err", err)
		return nil
	}
	log.Debug("文件读取", "b", b)
	sk := crypto.ToECDSAUnsafe(b)
	return sk
}

func (n *Node) RandomRtt() {
	count := 0
	var randomTime = []int64{1, 5, 10, 20, 1, 1,
		1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
		10, 12, 13, 14, 13, 12, 10, 10, 5, 5,
		5, 5, 10, 20, 25, 5, 5, 5, 5, 5}
	log.NewGoroutine(func() {
		for {
			common.Rtt = randomTime[count] * 10
			time.Sleep(5 * time.Second)
			count++
			if count >= len(randomTime) {
				break
			}
		}
	})
}
