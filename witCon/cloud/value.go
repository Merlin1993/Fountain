package cloud

const (
	stateRes = iota
	//传递节点信息，主要是nodeId和
	nodeInfo
	//获取exl文件
	nodeExl
)

const (
	//启动
	startReq = iota
	//停止
	stopReq
	//更新
	updateReq
	//collectExl
	collectReq

	//晶格链相关
	startZltcReq
	//更新genesis文件
	updateGenesisReq
	//更新配置
	updateConfigReq
	//清理缓存
	clearCacheReq
	//更新节点
	updateApp
)

const (
	pending uint = iota
	start
	stop
)

var (
	exit      = []byte("exit\n")
	Port      = 30005
	DefaultIP = "10.0.131.75"
	cmd       = "./node"
	//cmd            = "./node.exe"
	//zltcCmd        = "./capricorn.exe"
	configPath = "cfg.toml"
)
