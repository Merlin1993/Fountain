package common

type Config struct {
	Name               Address   `json:"name"`
	SaintList          []Address `json:"saintList"`
	IP                 string    `json:"IP"`
	NodeList           []string  `json:"nodeList"`
	LogLvl             uint      `json:"LogLvl"`
	Consensus          uint      `json:"Consensus"`
	TxAmount           uint      `json:"TxAmount"`
	TxSize             uint      `json:"TxSize"`
	ViewChangeDuration uint      `json:"ViewChangeDuration"`
	Rtt                uint      `json:"Rtt"`

	ShardVerify     bool   `json:"ShardVerify"`
	Schnorr         bool   `json:"SignatureVerify"`
	PrePack         bool   `json:"prePack"`
	SignVerifyCore  uint   `json:"signVerifyCore"`  //并行验证签名预计使用的核心数
	ShardCount      uint   `json:"shardCount"`      //分片的数量
	ShardVerifyCore uint   `json:"shardVerifyCore"` //分片并行验证使用的核心数
	VerifyNode      []uint `json:"verifyNode"`      //验证节点验证的分片
	VerifyCount     uint   `json:"verifyCount"`     //验证节点验证的分片数（可以同时验证多个分片，根据分片数和这个值确认最后会有多少个分片验证节点）

	TxPath string `json:"txpath"`
}

func (c *Config) Copy() *Config {
	return &Config{
		Name:               c.Name,
		SaintList:          c.SaintList,
		IP:                 c.IP,
		NodeList:           c.NodeList,
		LogLvl:             c.LogLvl,
		Consensus:          c.Consensus,
		TxAmount:           c.TxAmount,
		TxSize:             c.TxSize,
		ViewChangeDuration: c.ViewChangeDuration,
		Rtt:                c.Rtt,
		ShardVerify:        c.ShardVerify,
		Schnorr:            c.Schnorr,
		PrePack:            c.PrePack,
		SignVerifyCore:     c.SignVerifyCore,
		ShardCount:         c.ShardCount,
		ShardVerifyCore:    c.ShardVerifyCore,
		TxPath:             c.TxPath,
		VerifyNode:         c.VerifyNode,
		VerifyCount:        c.VerifyCount,
	}
}
