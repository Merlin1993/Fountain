package p2p

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/common/zerror"
	"witCon/log"
	"witCon/p2p/netutil"
)

var (
	defaultDialTimeout     = 15 * time.Second
	defaultMaxPendingPeers = 50
)

type Server struct {
	action          func()
	Dialer          *net.Dialer
	listener        net.Listener
	PS              *PeerSet
	name            common.Address
	proto           map[uint]ProtoHandle
	pendingNodeList map[string]*Node
	localNode       *Node
	maplock         sync.RWMutex
	quit            chan struct{}
	saintLen        int //用于判断是否有足够的saint满足action条件
	peerLen         int //用于判断多少个peer连接上可以action
}

type Node struct {
	IP   string
	Port int
}

func NewNode(ip string) *Node {
	ips := strings.SplitN(ip, ":", 2)
	port, _ := strconv.ParseInt(ips[1], 10, 0)
	return &Node{
		IP:   ips[0],
		Port: int(port),
	}
}

func (n *Node) ToString() string {
	return n.IP + ":" + fmt.Sprintf("%v", n.Port)
}

func NewServer(name common.Address, nodeList []string, IP string, peerLen int, saintLen int, action func()) *Server {
	srv := &Server{
		action:          action,
		Dialer:          nil,
		listener:        nil,
		PS:              NewPeerSet(),
		name:            name,
		proto:           make(map[uint]ProtoHandle),
		pendingNodeList: make(map[string]*Node),
		quit:            make(chan struct{}),
		localNode:       NewNode(IP),
		saintLen:        saintLen,
		peerLen:         peerLen,
	}
	log.Info(fmt.Sprintf("peerLen:%v", peerLen))
	log.Info(fmt.Sprintf("nodeList:%v", nodeList))
	for _, node := range nodeList {
		//如果有自己，排除掉
		if node == IP {
			continue
		}
		srv.pendingNodeList[node] = NewNode(node)
	}
	log.Info(fmt.Sprintf("pendingNodeList:%v", srv.pendingNodeList))
	srv.Dialer = &net.Dialer{Timeout: defaultDialTimeout}
	srv.setupListening()
	log.NewGoroutine(srv.dialLoop)
	return srv
}

func (srv *Server) RegisterProto(suite uint, handle ProtoHandle) {
	srv.proto[suite] = handle
}

func (srv *Server) dialLoop() {
	for {
		srv.maplock.Lock()
		for _, node := range srv.pendingNodeList {
			srv.dial(node)
		}
		srv.maplock.Unlock()
		time.Sleep(15 * time.Second)
	}
}

func (srv *Server) dial(dest *Node) error {
	log.Debug("dial", "ip", dest.IP, "port", dest.Port)
	addr := &net.TCPAddr{IP: net.ParseIP(dest.IP), Port: dest.Port}
	fd, err := srv.Dialer.Dial("tcp", addr.String())
	if err != nil {
		log.Error("dial fail", "ip", dest.IP, "port", dest.Port, "err", err)
		return err
	}
	return srv.SetupConn(fd, dest)
}

func (srv *Server) setupListening() error {
	log.Debug("start listen", "tcp", srv.localNode.ToString())
	listener, err := net.Listen("tcp", srv.localNode.ToString())
	if err != nil {
		log.Error("listen", "err", err)
		return err
	}
	srv.listener = listener

	go srv.listenLoop()
	return nil
}

func (srv *Server) listenLoop() {
	log.Debug("TCP listener up", "addr", srv.listener.Addr())

	tokens := defaultMaxPendingPeers
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
			fd, err = srv.listener.Accept()
			log.Debug("accept Fd")
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
		go func() {
			srv.SetupConn(fd, nil)
			slots <- struct{}{}
		}()
	}
}

func (srv *Server) SetupConn(quic net.Conn, dialDest *Node) error {
	if dialDest == nil {
		log.Debug("SetupConn", "ip", dialDest)
	} else {
		log.Debug("SetupConn", "ip", dialDest.ToString())
	}
	c := NewConn(quic)
	p := &Peer{
		conn: &c,
		node: dialDest,
	}
	p.Start(srv.name, func(code uint, data []byte) {
		suite, code := resolveProtocol(code)
		switch suite {
		case BaseSuite:
			switch code {
			case ConnectEOF:
				if p.node != nil {

					srv.maplock.Lock()
					srv.pendingNodeList[p.node.ToString()] = p.node
					srv.maplock.Unlock()
				}
				log.Error("rec EOF", "addr", quic.RemoteAddr())
			case HandshakeCode:
				hs := &Handshake{}
				err := rlp.DecodeBytes(data, hs)
				if err != nil {
					log.Error("rec HandshakeCode", "err", err)
				}
				log.Debug("rec HandshakeCode", "name", hs.Addr, "shard", fmt.Sprintf("%v", hs.Shards))
				p.name = hs.Addr
				p.shard = hs.Shards
				srv.PS.AddPeer(hs.Addr, p)
				if p.node != nil {
					srv.maplock.Lock()
					delete(srv.pendingNodeList, p.node.ToString())
					srv.maplock.Unlock()
				}
				if srv.PS.CheckAmount(srv.peerLen) {
					srv.PS.BroadcastAction()
					if srv.saintLen == 1 {
						srv.action()
					}
				}
			case RequestName:
				log.Debug("rec RequestName", "name", p.name)
				p.SendHandshake(srv.name, common.VerifyNode)
			case Action:
				log.Debug("rec action", "name", p.name)
				srv.maplock.Lock()
				srv.PS.actionMap[p.name] = struct{}{}
				if len(srv.PS.actionMap) == (srv.saintLen - 1) {
					srv.action()
				}
				srv.maplock.Unlock()
			}
		case ConsensusSuite:
			srv.proto[ConsensusSuite].HandleMsg(p.name, code, data)
		}

	})
	return nil
}

var (
	errInvalid     = zerror.New("不合法的ip地址", "invalid IP", 2901)
	errUnspecified = zerror.New("ip地址为0", "zero address", 2901)
	errSpecial     = zerror.New("special network", "special network", 2901)
	errLoopback    = zerror.New("ip地址开头为127", "loopback address from non-loopback host", 2901)
	errLAN         = zerror.New("special network", "LAN address from WAN host", 2901)
)
