package p2p

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
	"witCon/log"
)

var (
	client *HttpUtil
)

type HttpUtil struct {
	client *http.Client
}

func ReqSaintKey(url string) (saintkey string, err error) {
	body, err := ReqRPC(url, "wallet_saintKey", []interface{}{})
	if err != nil {
		log.Error("req saint key", "err", err)
		return "", err
	}
	r := SaintKeyResp{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		log.Error("req saint key unmarshal", "err", err)
		return "", err
	}
	return r.Result, nil
}

func ReqNodeInfo(url string) (nodeId string, err error) {
	body, err := ReqRPC(url, "node_nodeInfo", []interface{}{})
	if err != nil {
		log.Error("req node info", "err", err)
		return "", err
	}
	r := NodeInfoResp{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		log.Error("req node info unmarshal", "err", err)
		return "", err
	}
	return r.Result.INode, nil
}

func ReqPeerInfo(url string) (count int, err error) {
	body, err := ReqRPC(url, "node_peers", []interface{}{})
	if err != nil {
		log.Error("req node peer", "err", err)
		return 0, err
	}
	r := PeerInfoResp{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		log.Error("req node peer unmarshal", "err", err)
		return 0, err
	}
	return len(r.Result), nil
}

func ReqGetStatistic(url string) string {
	body, err := ReqRPC(url, "latc_getStatistic", []interface{}{})
	if err != nil {
		log.Error("req Statistic", "err", err)
		return ""
	}
	r := StatisticResp{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		log.Error("req Statistic", "err", err)
		return ""
	}
	return r.Result
}

func ReqRPC(url string, Method string, Params []interface{}) ([]byte, error) {
	if client == nil {
		client = &HttpUtil{client: &http.Client{Timeout: 25 * time.Second}}
	}
	return client.req("http://"+url, Method, Params)
}

func (f *HttpUtil) req(url string, Method string, Params []interface{}) ([]byte, error) {
	data := JsonRpcReq{
		Jsonrpc: "2.0",
		Method:  Method,
		Params:  Params,
		Id:      1,
	}
	jsonStr, _ := json.Marshal(data)
	resp, err := f.client.Post(url, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	return result, err
}

type JsonRpcReq struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type err struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type PeerInfoResp struct {
	JsonRpc string      `json:"jsonRpc"`
	Id      int         `json:"id"`
	Result  []*PeerInfo `json:"result"`
	Error   err         `json:"error"`
}

type PeerInfo struct {
	Inode   string   `json:"inode"` // Node URL
	ID      string   `json:"id"`    // Unique node identifier
	Name    string   `json:"name"`  // Name of the node, including client types, version, OS, custom data
	Caps    []string `json:"caps"`  // Protocols advertised by this peer
	Network struct {
		LocalAddress  string `json:"localAddress"`  // Local endpoint of the TCP data connection
		RemoteAddress string `json:"remoteAddress"` // Remote endpoint of the TCP data connection
		Inbound       bool   `json:"inbound"`
		Trusted       bool   `json:"trusted"`
		Static        bool   `json:"static"`
	} `json:"network"`
	Protocols map[string]interface{} `json:"protocols"` // Sub-protocol specific metadata fields
}

type NodeInfoResp struct {
	JsonRpc string    `json:"jsonRpc"`
	Id      int       `json:"id"`
	Result  *NodeInfo `json:"result"`
	Error   err       `json:"error"`
}

type NodeInfo struct {
	ID    string `json:"id"`    // Unique node identifier (also the encryption key)
	Name  string `json:"name"`  // Name of the node, including client types, version, OS, custom data
	INode string `json:"inode"` // INode URL for adding this peer from remote peers
	INR   string `json:"inr"`   // Node Record
	IP    string `json:"ip"`    // IP address of the node
	Ports struct {
		Discovery int `json:"discovery"` // UDP listening port for discovery protocol
		Listener  int `json:"listener"`  // TCP listening port for RLPx
	} `json:"ports"`
	ListenAddr string                 `json:"listenAddr"`
	Protocols  map[string]interface{} `json:"protocols"`
}

type SaintKeyResp struct {
	JsonRpc string `json:"jsonRpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
	Error   err    `json:"error"`
}

type StatisticResp struct {
	JsonRpc string `json:"jsonRpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
	Error   err    `json:"error"`
}

type NilResp struct {
	JsonRpc string `json:"jsonRpc"`
	Id      int    `json:"id"`
	Error   err    `json:"error"`
}
