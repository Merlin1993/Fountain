package rawbd

import (
	"witCon/common/zerror"
)

var (
	errMemoryDBClosed   = zerror.New("内存数据库已经关闭", "memory database closed", 2601)
	errMemoryDBNotFound = zerror.New("未发现内存数据库", "memory database not found", 2602)
)
