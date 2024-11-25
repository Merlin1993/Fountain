package stat

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/net"
	"sync"
	"time"
	"witCon/log"
)

var (
	Interval = 1 * time.Second
)

type SystemInfo struct {
	CpuTotalPercent    float64 `json:"cpuTotalPercent"`    //1s刷新一次
	AllCpuTotalPercent float64 `json:"allCpuTotalPercent"` //所有cpu的平均值
	MemTotalSize       uint64
	MemUsedSize        uint64
	MemUsedPercent     float64 `json:"memUsedPercent"`
	NetBytesRecv       uint64  `json:"netBytesRecv"`
	NetBytesSent       uint64  `json:"netBytesSent"`
	NetBytesRecvSpeed  uint64  `json:"netBytesRecvSpeed"`
	NetBytesSentSpeed  uint64  `json:"netBytesSentSpeed"`
	NetBytesRecvS      string  `json:"netBytesRecvS"`
	NetBytesSentS      string  `json:"netBytesSentS"`
	DiskWriteBytes     uint64  `json:"diskWriteBytes"`
	DiskReadBytes      uint64  `json:"diskReadBytes"`
	DiskUsedBytes      uint64  `json:"diskUsedBytes"`
	DiskTotalBytes     uint64  `json:"diskTotalBytes"`
	lastNetBytesRecv   uint64
	lastNetBytesSent   uint64
	lastDiskWriteBytes uint64
	lastDiskReadBytes  uint64
	lock               sync.RWMutex
	Path               string
	initFlag           bool "系统信息是否初始化过"
}

func (s *SystemInfo) mainLoop() {
	log.NewGoroutine(func() {
		for {
			cpuTotalPercent, err := cpu.Percent(1*time.Second, false)
			if err != nil {
				log.Error("get Cpu_total_percent_ err", "err", err)
				return
			}
			s.AllCpuTotalPercent = cpuTotalPercent[0]
		}
	})
	log.NewGoroutine(func() {
		for {
			info, err := net.IOCounters(false)
			if err != nil {
				log.Error("get net info err", "err", err)
				return
			}

			var Net_bytesRecv uint64 = 0
			var Net_bytesSent uint64 = 0
			for _, v := range info {
				Net_bytesRecv += v.BytesRecv
				Net_bytesSent += v.BytesSent
			}

			//开始采样。。。
			time.Sleep(1 * time.Second)
			info, err = net.IOCounters(false)
			if err != nil {
				log.Error("get net info err", "err", err)
				return
			}
			var Net_bytesRecv_ uint64 = 0
			var Net_bytesSent_ uint64 = 0
			for _, v := range info {
				Net_bytesRecv_ += v.BytesRecv
				Net_bytesSent_ += v.BytesSent
			}
			s.NetBytesRecvSpeed = Net_bytesRecv_ - Net_bytesRecv
			s.NetBytesSentSpeed = Net_bytesSent_ - Net_bytesSent
		}
	})

}
