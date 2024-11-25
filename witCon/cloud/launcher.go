package cloud

import (
	"fmt"
	"net"
	"path"
	"path/filepath"
	"sync"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/log"
	"witCon/p2p"
	"witCon/p2p/netutil"
)

//收集所有的节点ip
//然后告诉大家所有ip（大家这个时候就可以更改配置文件了）

// 方法
// 所有节点启动
// 所有节点更换文件
// 所有节点停止
type Launcher struct {
	listener               net.Listener
	defaultMaxPendingPeers int
	targetState            uint
	curState               uint
	pendingNode            map[string]*cloudNode
	readyNode              map[string]*cloudNode
	nodeIn                 map[string]*NodeInfo
	lstNode                []*cloudNode
	lock                   sync.RWMutex
	connect                bool
	zltcReady              bool
}

func NewLauncher(defaultMaxPendingPeers int, IP string) *Launcher {
	log.Create("", "launcher", 4)
	l := &Launcher{
		defaultMaxPendingPeers: defaultMaxPendingPeers,
		targetState:            stop,
		curState:               stop,
		pendingNode:            make(map[string]*cloudNode),
		readyNode:              make(map[string]*cloudNode),
		lstNode:                make([]*cloudNode, 0, defaultMaxPendingPeers),
		nodeIn:                 make(map[string]*NodeInfo),
	}
	l.startListen(IP)
	return l
}

func (l *Launcher) Start() {
	if l.checkConnectFail() || l.checkStopFail() {
		return
	}
	l.lock.Lock()
	defer l.lock.Unlock()
	l.target(start)
	l.broadcast(startReq)
}

func (l *Launcher) Stop() {
	if l.checkConnectFail() {
		return
	}
	l.target(stop)
	l.lock.Lock()
	defer l.lock.Unlock()
	l.broadcast(stopReq)
}

func (l *Launcher) Collect() {
	if l.checkConnectFail() {
		return
	}
	l.target(stop)
	l.lock.Lock()
	defer l.lock.Unlock()
	l.broadcast(collectReq)
}

func (l *Launcher) UpdateApp() {
	path := filepath.Clean("node")
	fdata := ReadFile(path)

	if len(fdata) == 0 {
		log.Error("find app fail")
		return
	}
	l.target(stop)
	l.broadcastSomething(updateApp, &APPFile{fdata})
}

func (l *Launcher) UpdateConfig() {
	if l.checkConnectFail() || l.checkStopFail() {
		return
	}
	l.lock.Lock()
	defer l.lock.Unlock()
	vn := make([]*cloudNode, 0)
	cn := make([]*cloudNode, 0)
	for _, ns := range l.lstNode {
		if l.nodeIn[ns.node.IP].Verify {
			vn = append(vn, ns)
		} else {
			cn = append(cn, ns)
		}
	}

	cfg := common.GetConfig(configPath)
	l.target(stop)
	l.pendingNode = l.readyNode
	l.readyNode = make(map[string]*cloudNode)

	//生成共识节点配置
	cnodeLen := len(cn)
	cconfigList := make([]*common.Config, cnodeLen)
	csaintList := make([]common.Address, cnodeLen)
	cnodeIP := make([]string, cnodeLen)
	for i := 0; i < cnodeLen; i++ {
		//name := fmt.Sprintf("%v", i)
		ipPort := 30000
		ips := fmt.Sprintf("%s:%v", cn[i].node.IP, ipPort)
		csaintList[i] = l.nodeIn[cn[i].node.IP].CommonAddr
		cnodeIP[i] = ips
		cconfigList[i] = cfg.Copy()
		cconfigList[i].Name = l.nodeIn[cn[i].node.IP].CommonAddr
		cconfigList[i].IP = ips
		cconfigList[i].VerifyNode = make([]uint, 0)
	}
	for i, v := range cn {
		cconfigList[i].SaintList = csaintList
		cconfigList[i].NodeList = cnodeIP[:i]
		log.Debug("send config", "config", cconfigList[i])
		v.conn.SendQuicMsg(updateReq, cconfigList[i])
	}

	//生成共识节点配置
	vnodeLen := len(vn)
	vconfigList := make([]*common.Config, vnodeLen)
	for i := 0; i < vnodeLen; i++ {
		//name := fmt.Sprintf("%v", i)
		ipPort := 30000
		ips := fmt.Sprintf("%s:%v", vn[i].node.IP, ipPort)
		vconfigList[i] = cfg.Copy()
		vconfigList[i].Name = l.nodeIn[vn[i].node.IP].CommonAddr
		vconfigList[i].IP = ips

		vsignVerifyCore := cfg.SignVerifyCore / cfg.ShardCount * cfg.VerifyCount
		if vsignVerifyCore < 2 {
			vsignVerifyCore = 2
		}
		vshardVerifyCore := cfg.SignVerifyCore / cfg.ShardCount * cfg.VerifyCount
		if vshardVerifyCore < 1 {
			vshardVerifyCore = 1
		}
		vconfigList[i].SignVerifyCore = vsignVerifyCore
		vconfigList[i].ShardVerifyCore = vshardVerifyCore

		vcn := make([]uint, cfg.VerifyCount)
		si := int(cfg.ShardCount) / vnodeLen
		for z := 0; z < int(cfg.VerifyCount); z++ {
			vcn[z] = uint(i*si+z) % cfg.ShardCount
		}
		vconfigList[i].VerifyNode = vcn
		vconfigList[i].NodeList = []string{cnodeIP[i%len(cnodeIP)]}

	}
	for i, v := range vn {
		log.Debug("send config", "config", vconfigList[i])
		v.conn.SendQuicMsg(updateReq, vconfigList[i])
	}
}

func (l *Launcher) broadcast(code uint) {
	l.pendingNode = l.readyNode
	l.readyNode = make(map[string]*cloudNode)
	for _, v := range l.lstNode {
		v.conn.SendQuicMsg(code, struct{}{})
	}
}

func (l *Launcher) broadcastSomething(code uint, data interface{}) {
	l.pendingNode = l.readyNode
	l.readyNode = make(map[string]*cloudNode)
	for _, v := range l.lstNode {
		v.conn.SendQuicMsg(code, data)
	}
}

func (l *Launcher) ClearCache() {
	if l.checkStopFail() || l.checkConnectFail() {
		return
	}
	l.lock.Lock()
	defer l.lock.Unlock()
	l.target(stop)
	l.zltcReady = false
	l.nodeIn = make(map[string]*NodeInfo)
	l.broadcast(clearCacheReq)
}

func (l *Launcher) checkConnectFail() bool {
	if !l.connect {
		fmt.Println("not all connect")
		return true
	}
	return false
}

func (l *Launcher) checkStartFail() bool {
	if l.curState != start {
		fmt.Println("state is not on start")
		return true
	}
	return false
}

func (l *Launcher) checkStopFail() bool {
	if l.curState != stop {
		fmt.Println("state is not on stop")
		return true
	}
	return false
}

func (l *Launcher) checkZltcReadyFail() bool {
	if !l.zltcReady {
		fmt.Println("zltc is not ready")
		return true
	}
	return false
}
func (l *Launcher) target(target uint) {
	l.curState = pending
	l.targetState = target
}

func (l *Launcher) startListen(IP string) {
	log.Debug("start listen", "port", Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", IP, Port))
	if err != nil {
		log.Error("listen", "err", err)
		return
	}
	l.listener = listener
	go l.listenLoop()
}

func (l *Launcher) listenLoop() {
	log.Debug("TCP listener up", "addr", l.listener.Addr())
	tokens := l.defaultMaxPendingPeers
	slots := make(chan struct{}, tokens)
	for i := 0; i < tokens; i++ {
		slots <- struct{}{}
	}

	for {
		<-slots
		var (
			fd  net.Conn
			err error
		)
		for {
			fd, err = l.listener.Accept()
			log.Debug("accept fd")
			if netutil.IsTemporaryError(err) {
				log.Debug("Temporary read error", "err", err)
				continue
			} else if err != nil {
				log.Debug("Read error", "err", err)
				return
			}
			break
		}
		//必须收集齐足够的链接才能启动

		log.Debug("receive conn", "addr", fd.RemoteAddr().String())

		l.SetupConn(fd)
		slots <- struct{}{}
	}
}

func (l *Launcher) SetupConn(fd net.Conn) {
	ip := fd.RemoteAddr().String()
	reacterNode := p2p.NewNode(ip)
	n := &cloudNode{
		conn: p2p.NewConn(fd),
		node: reacterNode,
	}
	exist := false
	for _, on := range l.lstNode {
		if on.node.IP == n.node.IP {
			log.Debug("find a old node")
			on.conn = n.conn
			on.node = n.node
			exist = true
			break
		}
	}
	if !exist {
		l.lstNode = append(l.lstNode, n)
	}
	l.readyNode[ip] = n
	if len(l.lstNode) == l.defaultMaxPendingPeers {
		fmt.Println("all node connect")
		l.connect = true
	}
	n.conn.ReadQuicMsgLoop(func(code uint, data []byte) {
		l.lock.Lock()
		defer l.lock.Unlock()
		switch code {
		case stateRes:
			var state uint
			err := rlp.DecodeBytes(data, &state)
			log.Debug("rec stateRes", "state", state)
			if err != nil {
				log.Error("rec stateRes", "err", err)
			}
			if l.curState != start && l.targetState == start && state == start {
				log.Info(fmt.Sprintf("%v change to start", ip))
				delete(l.pendingNode, ip)
				l.readyNode[ip] = n
				if len(l.pendingNode) == 0 {
					log.Info("all node change to start")
					l.curState = start
				}
			}
			if l.curState != stop && l.targetState == stop && state == stop {
				log.Info(fmt.Sprintf("%v change to stop", ip))
				delete(l.pendingNode, ip)
				l.readyNode[ip] = n
				if len(l.pendingNode) == 0 {
					log.Info("all node change to stop")
					l.curState = stop
				}
			}
		case nodeInfo:
			ni := &NodeInfo{}
			err := rlp.DecodeBytes(data, &ni)
			if err != nil {
				log.Error("rec nodeInfo", "err", err)
			}
			l.nodeIn[reacterNode.IP] = ni
			log.Debug("rec nodeInfo", "n", ni, "len", len(l.nodeIn))
			if len(l.nodeIn) == l.defaultMaxPendingPeers {
				l.zltcReady = true
				log.Debug("nodeInfo ready")
			}
		case nodeExl:
			ef := &ExcelFile{}
			err := rlp.DecodeBytes(data, &ef)
			if err != nil {
				log.Error("rec nodeExl", "err", err)
			}
			WriteFile(path.Clean(ef.Name), ef.FileData)
		}
	})
}

type NodeInfo struct {
	Addr       string
	NodeId     string
	CommonAddr common.Address
	Verify     bool
}

type cloudNode struct {
	conn p2p.Conn
	node *p2p.Node
}

type ExcelFile struct {
	Name     string
	FileData []byte
}

type APPFile struct {
	FileData []byte
}
