package log

import (
	"github.com/go-stack/stack"
	"github.com/petermattis/goid"
	"os"
	"time"
)

type Logger interface {
	New(ctx ...interface{}) Logger
	GetHandler() []Handler
	SetHandler(h []Handler)
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
}

type logger struct {
	ctx []interface{}
	h   *swapHandler
}

func (l *logger) New(ctx ...interface{}) Logger {
	child := &logger{newContext(l.ctx, ctx), new(swapHandler)}
	child.SetHandler([]Handler{l.h})
	return child
}

func newContext(prefix []interface{}, suffix []interface{}) []interface{} {
	normalizedSuffix := normalize(suffix)
	newCtx := make([]interface{}, len(prefix)+len(normalizedSuffix))
	n := copy(newCtx, prefix)
	copy(newCtx[n:], normalizedSuffix)
	return newCtx
}

func (l *logger) GetHandler() []Handler {
	return l.h.Get()
}

func (l *logger) SetHandler(h []Handler) {
	l.h.Swap(h)
}

func (l *logger) write(msg string, lvl Lvl, ctx []interface{}, skip int) {
	l.h.Log(&Record{
		Time: time.Now(),
		Lvl:  lvl,
		Msg:  msg,
		Ctx:  newContext(l.ctx, ctx),
		Call: stack.Caller(skip),
		Goid: goid.Get(),
		KeyNames: RecordKeyNames{
			Time: timeKey,
			Msg:  msgKey,
			Lvl:  lvlKey,
			Ctx:  ctxKey,
		},
	})
}

func (l *logger) Trace(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlTrace) {
		l.write(msg, LvlTrace, ctx, skipLevel)
	}
}

func (l *logger) Debug(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlDebug) {
		l.write(msg, LvlDebug, ctx, skipLevel)
	}
}

func (l *logger) Info(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlInfo) {
		l.write(msg, LvlInfo, ctx, skipLevel)
	}
}

func (l *logger) Warn(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlWarn) {
		l.write(msg, LvlWarn, ctx, skipLevel)
	}
}

func (l *logger) Error(msg string, ctx ...interface{}) {
	if LogLevel >= int(LvlError) {
		l.write(msg, LvlError, ctx, skipLevel)
	}
}

func (l *logger) Crit(msg string, ctx ...interface{}) {
	l.write(msg, LvlCrit, ctx, skipLevel)
	os.Exit(1)
}

func normalize(ctx []interface{}) []interface{} {
	// if the caller passed a Ctx object, then expand it
	if len(ctx) == 1 {
		if ctxMap, ok := ctx[0].(CtxMap); ok {
			ctx = ctxMap.toArray()
		}
	}

	// ctx needs to be even because it's a series of key/value pairs
	// no one wants to check for errors on logging functions,
	// so instead of erroring on bad input, we'll just make sure
	// that things are the right length and users can fix bugs
	// when they see the output looks wrong
	if len(ctx)%2 != 0 {
		ctx = append(ctx, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}

	return ctx
}

func Init(logLevel int, logKeep int64) {
	LogLevel = logLevel
	if logKeep != 0 {
		LogKeep = logKeep
	}
}
