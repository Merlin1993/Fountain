package common

import "witCon/common/zerror"

var (
	ErrAddrFormat = zerror.New("地址格式不合法", "address format is invalid", 3601)
)
