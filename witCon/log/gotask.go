package log

import (
	"fmt"
	"runtime/debug"
)

func NewGoroutine(task func()) {
	go func() {
		defer RecoverError()
		task()
	}()
}

func RecoverError() {
	if err := recover(); err != nil {
		//输出panic信息
		Error(fmt.Sprintf("%v", err))

		//输出堆栈信息
		Error(string(debug.Stack()))
	}
}
