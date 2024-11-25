package cloud

import (
	"bufio"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"time"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/crypto"
	"witCon/log"
	"witCon/p2p"
	"witCon/stat"
)

type Reactor struct {
	server  p2p.Conn
	connect bool
	node    *p2p.Node
	Dialer  *net.Dialer
	curCmd  *exec.Cmd
	stdin   io.WriteCloser
	isStart bool
	info    *NodeInfo
	addr    common.Address

	cfg *common.Config
}

func NewReactor(IP string, verify bool) *Reactor {
	log.Create("", "reactor", 4)
	if IP == "" {
		IP = DefaultIP
	}
	r := &Reactor{
		server:  p2p.Conn{},
		connect: false,
		node: &p2p.Node{
			IP:   IP,
			Port: Port,
		},
	}
	r.Dialer = &net.Dialer{Timeout: 15 * time.Second}
	log.Debug(fmt.Sprintf("start with %v:%v", IP, Port))
	r.addr = r.CreateSK()
	r.info = &NodeInfo{
		Addr:       "",
		NodeId:     "",
		CommonAddr: r.addr,
		Verify:     verify,
	}
	go r.dialLoop()
	return r
}

func (r *Reactor) dialLoop() {
	for {
		if !r.connect {
			log.Info("try dial")
			r.dial(r.node)
		}
		time.Sleep(15 * time.Second)
	}
}

func (r *Reactor) dial(dest *p2p.Node) error {
	log.Debug("dial", "ip", dest.IP, "port", dest.Port)
	addr := &net.TCPAddr{IP: net.ParseIP(dest.IP), Port: dest.Port}
	fd, err := r.Dialer.Dial("tcp", addr.String())
	if err != nil {
		return err
	}
	r.SetupConn(fd)
	return nil
}

func (r *Reactor) SetupConn(fd net.Conn) {
	r.connect = true
	r.server = p2p.NewConn(fd)
	r.server.ReadQuicMsgLoop(func(code uint, data []byte) {
		switch code {
		//case p2p.ConnectEOF:
		//	r.connect = false
		case startReq:
			r.start()
		case stopReq:
			r.stop()
		case collectReq:
			r.sendExl()
		case updateApp:
			var appfile = &APPFile{}
			err := rlp.DecodeBytes(data, appfile)
			if err != nil {
				log.Error("rec updateAPP", "err", err)
			}
			log.Debug("rec updateApp")
			r.updateApp(appfile.FileData)
		case updateReq:
			var cfg = &common.Config{}
			err := rlp.DecodeBytes(data, cfg)
			log.Debug("rec updateReq", "config", cfg)
			if err != nil {
				log.Error("rec updateReq", "err", err)
			}
			r.updateConfig(cfg)
		case p2p.ConnectEOF:
			return
		}
	})
	log.Info("send nodeInfo", "info", r.info)
	r.server.SendQuicMsg(nodeInfo, r.info)
}

func (r *Reactor) start() {
	r.isStart = true
	command := cmd
	params := []string{"start"}
	log.Info("start")
	go r.execCommand(command, params, true)
}

// todo stop 没有效果
func (r *Reactor) stop() {
	r.isStart = false
	if r.curCmd == nil {
		if r.server.Fd == nil {
			return
		}
		r.server.SendQuicMsg(stateRes, stop)
		return
	}
	r.stdin.Write(exit)
	if r.curCmd.ProcessState != nil {
		pid := r.curCmd.ProcessState.Pid()
		log.Error("stop kill", "pid", pid)
		r.execCommand("kill", []string{"-9", fmt.Sprintf("%v", pid)}, false)
	}

	log.Info("stop")
	if r.server.Fd == nil {
		return
	}
	r.server.SendQuicMsg(stateRes, stop)
}

func (r *Reactor) sendExl() {
	name := stat.GetExlFile(r.cfg)
	path := filepath.Clean(name)
	fdata := ReadFile(path)
	if len(fdata) != 0 {
		ef := &ExcelFile{
			Name:     name,
			FileData: fdata,
		}
		r.server.SendQuicMsg(nodeExl, ef)
	} else {
		log.Error("exl file not found")
	}
	r.server.SendQuicMsg(stateRes, stop)
}

func (r *Reactor) updateConfig(cfg *common.Config) {
	//写入到文件
	configPath := common.DefaultConfigPath
	file, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777) //WRONLY，清空
	if err != nil {
		log.Error("文件打开失败", "err", err)
		return
	}
	defer file.Close()

	//文件写入缓冲区
	wri := bufio.NewWriter(file)
	encoder := toml.NewEncoder(wri)
	encoder.Encode(cfg)
	wri.Flush()
	r.cfg = cfg
	r.server.SendQuicMsg(stateRes, stop)
}

// todo 有批量重装系统的操作，暂时来看没必要
func (r *Reactor) updateApp(data []byte) {
	WriteFile(path.Clean("node"), data)
	r.server.SendQuicMsg(stateRes, stop)
}

func RecoverError() {
	if err := recover(); err != nil {
		//输出panic信息
		log.Error(fmt.Sprintf("%v", err))

		//输出堆栈信息
		log.Error(string(debug.Stack()))
	}
}

func (r *Reactor) execCommand(commandName string, params []string, isStart bool) bool {
	defer RecoverError()
	r.curCmd = exec.Command(commandName, params...)

	//显示运行的命令
	log.Debug("exec command", "arg", r.curCmd.Args)

	stdout, err := r.curCmd.StdoutPipe()
	r.stdin, err = r.curCmd.StdinPipe()
	stderr, err := r.curCmd.StderrPipe()
	if err != nil {
		log.Error(err.Error())
		return false
	}

	err = r.curCmd.Start()
	if isStart {
		r.server.SendQuicMsg(stateRes, start)
	}
	if err != nil {
		log.Error("start fail", "err", err.Error())
		return false
	}
	reader := bufio.NewReader(stdout)
	errReader := bufio.NewReader(stderr)
	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			log.Debug("end", "err", err2)
			if r.isStart {
				log.Error("start with stop")
				for {
					line2, err3 := errReader.ReadString('\n')
					if err3 != nil || io.EOF == err3 {
						break
					}
					log.Error(line2)
				}
				r.execCommand(commandName, params, isStart)
			}
			break
		}
		fmt.Print(" : " + line)
	}

	r.curCmd.Wait()
	return true
}

// 为共识算法生成一个私钥
func (r *Reactor) CreateSK() common.Address {
	skpath := common.DefaultSKPath
	common.DataPath = skpath
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
