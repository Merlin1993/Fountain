package p2p

import (
	"testing"
	"witCon/common"
)

func TestEncode(t *testing.T) {
	Encode(common.BCPayload)
}

func TestProtocolCode(t *testing.T) {
	x := mixProtocol(1, 2)
	t.Log(x)

	y, z := resolveProtocol(x)
	t.Log(y)
	t.Log(z)

}
