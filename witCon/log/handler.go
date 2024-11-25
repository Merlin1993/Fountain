package log

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"github.com/onsi/ginkgo/reporters/stenographer/support/go-colorable"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Handler interface {
	Log(r *Record) error
}

type LogFileWriter struct {
	Name string
	os.File
}

type funcHandler func(r *Record) error

func (h funcHandler) Log(r *Record) error {
	return h(r)
}

func FuncHandler(fn funcHandler) Handler {
	return fn
}

// StreamHandler writes log records to an io.Writer with the given format.
// StreamHandler can be used to easily begin writing log records to other outputs.
func StreamHandler(wr io.Writer, fmtR Format) Handler {
	h := FuncHandler(func(r *Record) error {
		_, err := wr.Write(fmtR.Format(r))
		return err
	})
	return LazyHandler(SyncHandler(h))
}

func SyncHandler(h Handler) Handler {
	var mu sync.Mutex
	return FuncHandler(func(r *Record) error {
		defer mu.Unlock()
		mu.Lock()
		return h.Log(r)
	})
}

// FileHandler returns a handler which writes log records to the give file
// using the given format. If the path
// already exists, FileHandler will append to the given file. If it does not,
// FileHandler will create the file with mode 0644.
func FileHandler(path string, name string, fmtr Format) (Handler, error) {

	f, err := generateLogFile(path, name)
	if err != nil {
		return nil, err
	}
	lfw := LogFileWriter{
		Name: name,
		File: *f,
	}
	go lfw.logLoop(path)
	return closingHandler{&lfw, StreamHandler(&lfw.File, fmtr)}, nil
}

func generateLogFile(path string, name string) (*os.File, error) {
	//判断path对应文件目录是否存在
	//若目录不存在，则创建
	_, err := os.Stat(path)
	if err != nil {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return nil, err
		}
	}
	//"2006-01-02-15-04-05"
	// 日志文件名由时间和saintaddr生成
	fileName := name + "_" + time.Now().Format("2006-01-02") + ".log"
	logPath := path + "/" + fileName
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (lfw *LogFileWriter) logLoop(path string) {
	for range time.Tick(5 * time.Minute) {
		lfw.updateLogFile(path)
		freshLogFiles(path)
	}
}

func (lfw *LogFileWriter) updateLogFile(path string) {
	f, err := generateLogFile(path, lfw.Name)
	if err != nil {
		return
	}
	lfw.File = *f
}

func freshLogFiles(logPath string) {
	//日志文件全保留
	if LogKeep == -1 {
		return
	}

	filepathNames, err := filepath.Glob(filepath.Join(logPath, "*"))
	if err != nil {
		fmt.Println("Failed to get the log file list!", "err", err)
	}

	//去除列表中的目录
	for i := 0; i < len(filepathNames); {
		if s, err := os.Stat(filepathNames[i]); err == nil && s.IsDir() {
			filepathNames = append(filepathNames[:i], filepathNames[i+1:]...)
		} else {
			i++
		}
	}

	//去除不满足日志文件格式的文件，不处理这些文件
	//已知日志文件格式：zltc_Qq87X3VBjGCTp3dcedxyQXadrTkmWbX1s_2021-10-12.log
	//或者NoSaint_2021-10-26.log
	for i := 0; i < len(filepathNames); {
		match1, _ := regexp.MatchString("zltc_[A-Za-z0-9]{12,}_\\d{4}-\\d{1,2}-\\d{1,2}.log", filepath.Base(filepathNames[i]))
		match2, _ := regexp.MatchString("NoSaint_\\d{4}-\\d{1,2}-\\d{1,2}.log", filepath.Base(filepathNames[i]))
		if match1 == false && match2 == false {
			filepathNames = append(filepathNames[:i], filepathNames[i+1:]...)
		} else {
			i++
		}
	}

	for i := range filepathNames {
		//已知日志文件格式：zltc_Qq87X3VBjGCTp3dcedxyQXadrTkmWbX1s_2021-10-12.log
		//获得log文件创建时间
		separate := strings.Split(strings.TrimRight(filepath.Base(filepathNames[i]), ".log"), "_")
		date := separate[len(separate)-1]

		//获得当前日期
		loc, _ := time.LoadLocation("Local") //获取时区
		timeLayout := "2006-01-02"           //转化所需模板
		now := time.Now().Format(timeLayout)
		//将日期转为Time格式
		tmp1, _ := time.ParseInLocation(timeLayout, date, loc) //log文件创建时间
		tmp2, _ := time.ParseInLocation(timeLayout, now, loc)  //当前时间
		timestamp1 := tmp1.Unix()                              //转化为时间戳 类型是int64
		timestamp2 := tmp2.Unix()                              //转化为时间戳 类型是int64
		day := (timestamp2 - timestamp1) / 86400               //除以一天的秒数

		//时间差大于LogKeep，则删除文件
		if day > LogKeep {
			err := os.Remove(filepathNames[i])
			if err != nil {
				fmt.Println("Failed to delete expired log files! ", "err", err)
			}
		}
	}
}

// XXX: closingHandler is essentially unused at the moment
// it's meant for a future time when the Handler interface supports
// a possible Close() operation
type closingHandler struct {
	io.WriteCloser
	Handler
}

// LazyHandler writes all values to the wrapped handler after evaluating any lazy functions in the record's context.
// It is already wrapped around StreamHandler and SyslogHandler in this library, you'll only need
// it if you write your own Handler.
//LazyHandler在对record上下文中的任何惰性函数求值后，将所有值写入包装的处理程序。
func LazyHandler(h Handler) Handler {
	return FuncHandler(func(r *Record) error {
		hadErr := false
		for i := 1; i < len(r.Ctx); i += 2 {

		}

		if hadErr {
			r.Ctx = append(r.Ctx, errorKey, "bad lazy")
		}
		return h.Log(r)
	})
}

func DiscardHandler() Handler {
	return FuncHandler(func(r *Record) error {
		return nil
	})
}

func TerminalHandlerColor() Handler {
	useColor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stdout)
	if useColor {
		output = colorable.NewColorableStdout()
	}
	return StreamHandler(output, TerminalFormat(useColor))
}

func LvlFilterHandler(maxLvl Lvl, h Handler) Handler {
	return FilterHandler(func(r *Record) (pass bool) {
		return r.Lvl <= maxLvl
	}, h)
}

func FilterHandler(fn func(r *Record) bool, h Handler) Handler {
	return FuncHandler(func(r *Record) error {
		if fn(r) {
			return h.Log(r)
		}
		return nil
	})
}
