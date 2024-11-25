package main

import (
	"bufio"
	"fmt"
	"github.com/naoina/toml"
	"gopkg.in/urfave/cli.v1"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"witCon/cloud"
	"witCon/common"
	"witCon/console"
	"witCon/crypto"
	"witCon/log"
	"witCon/node/utils"
)

var (
	app = NewApp("launcher command line interface")
)

func NewApp(usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = "snz"
	app.Usage = usage
	return app
}

func init() {
	app.Action = launcher
	app.HideVersion = true
	app.Copyright = "Copyright 2023"
	app.Commands = []cli.Command{
		{
			Action:      utils.MigrateFlags(launcher),
			Name:        "start",
			Usage:       "Start an interactive JavaScript environment with config",
			Flags:       []cli.Flag{utils.TypeFlag, utils.CountFlag},
			Category:    "CONSOLE COMMANDS",
			Description: ``,
		},
	}
	app.Flags = append(app.Flags, utils.TypeFlag, utils.CountFlag, utils.IPFlag)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func launcher(ctx *cli.Context) error {
	console, err := console.NewConsole()
	fmt.Println("console start")
	if err != nil {
		fmt.Println(err)
	}
	IP := ctx.GlobalString(utils.IPFlag.Name)
	if t := ctx.GlobalString(utils.TypeFlag.Name); t != "" {
		switch t {
		case "launcher":
			count := ctx.GlobalInt(utils.CountFlag.Name)
			makeLauncher(count, IP)
		case "reactor":
			makeReactor(IP)
		case "verify":
			makeVerifyReactor(IP)
		case "local":
			makeLocal()
		}
	}
	console.Interactive()
	return nil
}

func makeReactor(IP string) {
	//localIp, err := netutil.LocalIPv4s()
	//fmt.Print(localIp)
	//if len(localIp) > 0 && err == nil {
	//	for _, ip := range localIp {
	//		break
	//	}
	//}
	cloud.NewReactor(IP, false)
	<-make(chan struct{})
}

func makeVerifyReactor(IP string) {
	//localIp, err := netutil.LocalIPv4s()
	//fmt.Print(localIp)
	//if len(localIp) > 0 && err == nil {
	//	for _, ip := range localIp {
	//		break
	//	}
	//}
	cloud.NewReactor(IP, true)
	<-make(chan struct{})
}

func makeLauncher(count int, IP string) {
	l := cloud.NewLauncher(count, IP)
	for {
		cmd, err := console.Stdin.Prompt("next cmd: ")
		if err != nil {

		}
		switch cmd {
		case "start":
			l.Start()
		case "stop":
			l.Stop()
		case "update":
			l.UpdateConfig()
		case "collect":
			l.Collect()
		case "app":
			l.UpdateApp()
		case "clear":
			l.ClearCache()
		case "exit":
			return
		}
	}
}

// 本地运行数个节点
func makeLocal() {
	lpath, err := os.Getwd()
	if err != nil {
		log.Crit("get wd fail", "err", err)
	}
	lcfg := GetLocalConfig(lpath + "/local.toml")
	var path = lcfg.Path
	var nodePath = lcfg.NodePath
	var IP = lcfg.IP
	var nodeName = lcfg.NodeName
	var (
		nodes              = int(lcfg.Nodes)         //节点数
		logLvl             = lcfg.LogLvl             //日志等级，3为只显示重要信息，4表示详细信息
		consensus          = lcfg.Consensus          //共识算法 wit 0 , pbft 1, hotstuff 2, jolteon 3
		txAmount           = lcfg.TxAmount           //每个区块中交易数量 size
		txSize             = lcfg.TxSize             //每笔交易的大小（byte） 1024 = 1kB
		viewChangeDuration = lcfg.ViewChangeDuration //viewChange等待时间
		rtt                = lcfg.Rtt
		prePack            = lcfg.PrePack //是否提前打包，形成流水线
		shardVerify        = lcfg.ShardVerify
		schnorr            = lcfg.Schnorr
		signVerifyCore     = lcfg.SignVerifyCore
		shardCount         = lcfg.ShardCount
		shardVerifyCore    = lcfg.ShardVerifyCore
		txPath             = lcfg.TxPath
		verifyCount        = lcfg.VerifyCount
	)
	//var path = "E:\\wit-cons\\0910"
	//var nodePath = "D:\\go_workspace\\witCon"
	//var IP = "192.168.1.52"
	//var nodeName = "node.exe"
	//var (
	//	nodes              = 4     //节点数
	//	logLvl             = 3     //日志等级，3为只显示重要信息，4表示详细信息
	//	consensus          = 1     //共识算法 symphony 0 , pbft 1, hotstuff 2，jolteon 3
	//	txAmount           = 400   //每个区块中交易数量 size
	//	txSize             = 10000 //每笔交易的大小（byte） 1024 = 1kB
	//	viewChangeDuration = 50    //viewChange等待时间
	//	rtt                = 0
	//)
	nodeFile := filepath.Join(nodePath, nodeName)
	configList := make([]*common.Config, nodes)
	saintList := make([]common.Address, nodes)
	nodeIP := make([]string, nodes)
	addr := make([]common.Address, nodes)
	for i := 0; i < nodes; i++ {
		nPath := filepath.Join(path, strconv.Itoa(i))
		err := os.Mkdir(nPath, 0666)
		if err != nil {
			//fmt.Println("mkdir err", err)
		}
		common.DataPath = nPath
		addr[i] = CreateSK(nPath)
	}

	for i := 0; i < nodes; i++ {
		ipPort := 30000 + i
		ips := fmt.Sprintf("%s:%v", IP, ipPort)
		saintList[i] = addr[i]
		nodeIP[i] = ips
		configList[i] = &common.Config{
			LogLvl:             uint(logLvl),
			Consensus:          uint(consensus),          //symphony 0 , pbft 1, hotstuff 2，jolteon 3
			TxAmount:           uint(txAmount),           //size
			TxSize:             uint(txSize),             //1024 = 1kB
			ViewChangeDuration: uint(viewChangeDuration), //viewChange等待时间
			Rtt:                uint(rtt),                //viewChange等待时间
			ShardVerify:        shardVerify,
			PrePack:            prePack,
			Schnorr:            schnorr,
			SignVerifyCore:     signVerifyCore,
			ShardCount:         shardCount,
			ShardVerifyCore:    shardVerifyCore,
			VerifyNode:         []uint{},
			VerifyCount:        verifyCount,
			TxPath:             txPath,
		}
		configList[i].Name = addr[i]
		configList[i].IP = ips
	}

	//新建验证节点
	vconfigList := make([]*common.Config, shardCount)
	vaddr := make([]common.Address, shardCount)
	if verifyCount != 0 {
		for i := 0; i < int(shardCount); i++ {
			nPath := filepath.Join(path, "shard_"+strconv.Itoa(i))
			err := os.Mkdir(nPath, 0666)
			if err != nil {
				//fmt.Println("mkdir err", err)
			}
			common.DataPath = nPath
			vaddr[i] = CreateSK(nPath)
		}
		for i := 0; i < int(shardCount); i++ {
			ipPort := 40000 + i
			ips := fmt.Sprintf("%s:%v", IP, ipPort)

			vn := make([]uint, verifyCount)
			for z := 0; z < int(verifyCount); z++ {
				vn[z] = uint(i+z) % shardCount
			}
			vsignVerifyCore := signVerifyCore / shardCount * verifyCount
			if vsignVerifyCore < 2 {
				vsignVerifyCore = 2
			}
			vshardVerifyCore := signVerifyCore / shardCount * verifyCount
			if vshardVerifyCore < 1 {
				vshardVerifyCore = 1
			}
			vconfigList[i] = &common.Config{
				LogLvl:             uint(logLvl),
				Consensus:          uint(consensus),          //symphony 0 , pbft 1, hotstuff 2，jolteon 3
				TxAmount:           uint(txAmount),           //size
				TxSize:             uint(txSize),             //1024 = 1kB
				ViewChangeDuration: uint(viewChangeDuration), //viewChange等待时间
				Rtt:                uint(rtt),                //viewChange等待时间
				ShardVerify:        true,
				PrePack:            prePack,
				Schnorr:            schnorr,
				SignVerifyCore:     vsignVerifyCore,
				ShardCount:         shardCount,
				ShardVerifyCore:    vshardVerifyCore,
				VerifyNode:         vn,
				VerifyCount:        verifyCount,
				TxPath:             txPath,
			}
			vconfigList[i].IP = ips
			vconfigList[i].Name = vaddr[i]
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < nodes; i++ {
		configList[i].SaintList = saintList
		configList[i].NodeList = nodeIP[:i]
		nPath := filepath.Join(path, strconv.Itoa(i))
		write(nPath, configList[i])
		nFile := filepath.Join(nPath, nodeName)
		CopyFile(nodeFile, nFile)
		configPath := filepath.Join(nPath, common.DefaultConfigPath)
		command := filepath.Join(nPath, nodeName) //"node.exe"
		params := []string{"start", "-config", configPath, "-dataDir", fmt.Sprintf("%s\\", nPath)}
		//执行cmd命令
		wg.Add(1)
		go execCommand(configList[i].Name.String(), command, params)
	}
	if verifyCount != 0 {
		for i := 0; i < int(shardCount); i++ {
			vconfigList[i].NodeList = []string{nodeIP[i%len(nodeIP)]}
			nPath := filepath.Join(path, "shard_"+strconv.Itoa(i))
			write(nPath, vconfigList[i])
			nFile := filepath.Join(nPath, nodeName)
			//拷贝运行程序
			CopyFile(nodeFile, nFile)
			configPath := filepath.Join(nPath, common.DefaultConfigPath)
			command := filepath.Join(nPath, nodeName) //"node.exe"
			params := []string{"start", "-config", configPath, "-dataDir", fmt.Sprintf("%s\\", nPath)}
			//执行cmd命令
			wg.Add(1)
			go execCommand(vconfigList[i].Name.String(), command, params)
		}
	}
	wg.Wait()
}

func write(path string, config *common.Config) {
	configPath := fmt.Sprintf("%s/%v", path, common.DefaultConfigPath)
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_TRUNC, 0666) //WRONLY，清空
	if err != nil {
		fmt.Println("文件打开失败", err)
		return
	}
	defer file.Close()

	//文件写入缓冲区
	wri := bufio.NewWriter(file)
	encoder := toml.NewEncoder(wri)
	encoder.Encode(config)
	wri.Flush()
}

func CreateSK(path string) common.Address {
	skpath := filepath.Join(path, common.DefaultSKPath)
	fmt.Println("create sk", "skpath", skpath)
	if _, err := os.Stat(skpath); os.IsNotExist(err) {
		sk, err := crypto.GenerateKey()
		if err != nil {
			fmt.Println("私钥生成失败", err)
			return common.Address{}
		}
		b := crypto.FromECDSA(sk)
		fmt.Println("私钥生成", b)

		err = ioutil.WriteFile(skpath, b, 0777)
		if err != nil {
			fmt.Println("文件写入失败", err)
			return common.Address{}
		}
		fmt.Println("文件写入成功", err)
		return crypto.PubKeyToAddress(sk.PublicKey)
	} else {
		b, err := ioutil.ReadFile(skpath)
		if err != nil {
			log.Error("文件读取失败", "err", err)
			return common.Address{}
		}
		log.Error("文件读取", "b", b)
		sk := crypto.ToECDSAUnsafe(b)
		return crypto.PubKeyToAddress(sk.PublicKey)
	}
}

func CopyFile(dstName, srcName string) (writeen int64, err error) {
	src, err := os.Open(dstName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(srcName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func execCommand(id, commandName string, params []string) bool {
	cmd := exec.Command(commandName, params...)

	//显示运行的命令
	fmt.Println(cmd.Args)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		fmt.Println(err)
		return false
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
		return false
	}

	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fmt.Print(id + " : " + line)
	}

	cmd.Wait()
	return true
}
