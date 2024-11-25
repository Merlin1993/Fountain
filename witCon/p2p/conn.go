package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"time"
	"witCon/common"
	"witCon/common/rlp"
	"witCon/log"
	"witCon/stat"
)

var (
	quicLen = 1
)

type Conn struct {
	Fd net.Conn
	mc chan *msgEvent
}

func NewConn(fd net.Conn) Conn {
	c := Conn{
		Fd: fd,
		mc: make(chan *msgEvent, 100000),
	}
	log.NewGoroutine(func() {
		c.MsgLoop()
	})
	return c
}

type msgEvent struct {
	code uint
	data interface{}
	time int64
}

type QuicMsg struct {
	Code uint64
	Data []byte
}

func (p *Conn) MsgLoop() {
	for {
		select {
		case ev := <-p.mc:
			now := time.Now().UnixMilli()
			if now-ev.time < common.Rtt {
				time.Sleep(time.Duration(common.Rtt+ev.time-now) * time.Millisecond)
			}
			p.sendQuicMsg(ev.code, ev.data)
		}
	}
}
func (p *Conn) SendQuicMsg(code uint, data interface{}) error {
	ev := &msgEvent{
		code: code,
		data: data,
		time: time.Now().UnixMilli(),
	}
	p.mc <- ev
	return nil
}

func (p *Conn) sendQuicMsg(code uint, data interface{}) error {
	size, r, err := rlp.EncodeToReader(data)
	if err != nil {
		return err
	}
	buffer := make([]byte, size)
	r.Read(buffer)
	return p.sendMsg(code, buffer)
}

func (p *Conn) SendPlainQuicMsg(code uint, buffer []byte) error {
	return p.sendMsg(code, buffer)
}

func (p *Conn) sendMsg(code uint, buffer []byte) error {
	msg := &QuicMsg{
		Code: uint64(code),
		Data: buffer,
	}
	size, r, err := rlp.EncodeToReader(msg)
	if err != nil {
		return err
	}
	b2 := make([]byte, size)
	r.Read(b2)
	stat.Instance.OnFdWrite(size)
	log.Trace("quic send", "code", code, "data", len(b2), "size", size)
	b3, err := Encode(b2)
	if err != nil {
		return err
	}
	p.Fd.Write(b3)
	return nil
}

func (p *Conn) ReadQuicMsgLoop(handle func(code uint, data []byte)) {
	go func() {
		defer p.Fd.Close()
		reader := bufio.NewReader(p.Fd)
		for {
			nBytes, err := Decode(reader)
			if err != nil {
				if err != io.EOF {
					log.Error("quic read error", "err", err)
				}
				handle(ConnectEOF, []byte{})
				return
			}
			stat.Instance.OnFdRead(len(nBytes))
			log.Trace("quic receive nBytes", "nBytes", nBytes)
			msg := &QuicMsg{}
			err = rlp.DecodeBytes(nBytes, msg)
			if err == nil {
				handle(uint(msg.Code), msg.Data)
			} else {
				//log.Error("quic receive err", "err", err, "nBytes", nBytes)
			}
		}
	}()
}

func Encode(data []byte) ([]byte, error) {
	// 读取消息的长度转换成int32类型（4字节）
	var length = int32(len(data))
	var pkg = new(bytes.Buffer)
	// 写入消息头
	err := binary.Write(pkg, binary.LittleEndian, length)
	if err != nil {
		return nil, err
	}
	// 写入包体
	err = binary.Write(pkg, binary.LittleEndian, data)
	if err != nil {
		return nil, err
	}
	return pkg.Bytes(), nil
}

// 解码
func Decode(reader *bufio.Reader) ([]byte, error) {
	// 读消息长度
	lengthByte, _ := reader.Peek(4)
	lengthBuff := bytes.NewBuffer(lengthByte)
	var length int32
	err := binary.Read(lengthBuff, binary.LittleEndian, &length)
	if err != nil {
		log.Error("read length", "err", err)
		return nil, err
	}
	pack := make([]byte, int(4+length))
	all := length + 4
	var count int32 = 0
	for {
		// buffer返回缓冲中现有的可读的字节数
		n, err := reader.Read(pack[count:])
		if err != nil {
			log.Error("read buf", "err", err)
			return nil, err
		}
		count += int32(n)
		log.Trace("read count", "count", count, "all", all)
		if count == all {
			break
		}
		// 读取真正的数据
	}
	return pack[4:], nil
}
