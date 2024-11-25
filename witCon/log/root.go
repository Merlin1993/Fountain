package log

import (
	"os"
)

var (
	root = &logger{[]interface{}{}, new(swapHandler)}
)
var l Logger

func Create(path string, name string, logLevel int) {
	if path == "" {
		path = "data"
	}
	if logLevel == 0 {
		LogLevel = 4
	} else {
		LogLevel = logLevel
	}
	handler, err := FileHandler(path, name, LogFmtFormat())
	if err != nil {
		root.SetHandler([]Handler{TerminalHandlerColor()})
		root.Error("error path", "err", err)
	} else {
		root.SetHandler([]Handler{handler, TerminalHandlerColor()})
	}
}

func New(ctx ...interface{}) Logger {
	return root.New(ctx...)
}

func Root() Logger {
	return root
}

func Trace(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlTrace) {
		root.write(msg, LvlTrace, ctx, skipLevel)
	}
}

func Debug(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlDebug) {
		root.write(msg, LvlDebug, ctx, skipLevel)
	}
}

func Info(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlInfo) {
		root.write(msg, LvlInfo, ctx, skipLevel)
	}
}

func Warn(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlWarn) {
		root.write(msg, LvlWarn, ctx, skipLevel)
	}
}

func Error(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlError) {
		root.write(msg, LvlError, ctx, skipLevel)
	}
}

func Crit(msg string, ctx ...interface{}) {
	root.write(msg, LvlCrit, ctx, skipLevel)
	os.Exit(1)
}

func Output(msg string, lvl Lvl, callDepth int, ctx ...interface{}) {
	root.write(msg, lvl, ctx, callDepth+skipLevel)
}
