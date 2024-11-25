package log

import (
	"github.com/go-stack/stack"
	"time"
)

type RecordKeyNames struct {
	Time string
	Msg  string
	Lvl  string
	Ctx  string
}

type Record struct {
	Time     time.Time
	Lvl      Lvl
	Msg      string
	Ctx      []interface{}
	Call     stack.Call
	Goid     int64
	KeyNames RecordKeyNames
}
