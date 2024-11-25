package netutil

import (
	"witCon/common/zerror"
)

//todo 不知道怎么描述@robin
var (
	errInvalid     = zerror.New("不合法的ip地址", "invalid IP", 2901)
	errUnspecified = zerror.New("ip地址为0", "zero address", 2901)
	errSpecial     = zerror.New("special network", "special network", 2901)
	errLoopback    = zerror.New("ip地址开头为127", "loopback address from non-loopback host", 2901)
	errLAN         = zerror.New("special network", "LAN address from WAN host", 2901)
)

func IsTemporaryError(err error) bool {
	tempErr, ok := err.(interface {
		Temporary() bool
	})
	return ok && tempErr.Temporary() || isPacketTooBig(err)
}
