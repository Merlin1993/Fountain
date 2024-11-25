package stat

import (
	"os"
	"witCon/log"

	"github.com/shirou/gopsutil/process"

	//"reflect"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"

	//"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type Info struct {
	Cpu_total_percent      float64 //3s刷新一次
	All_Cpu_total_percent  float64
	Mem_usedPercent        float64
	Mem_totalSize          uint64
	Mem_usedSize           uint64
	Net_bytesRecv          uint64
	Net_bytesSent          uint64
	Net_bytesRecv_speed    uint64
	Net_bytesSent_speed    uint64
	Disk_writeBytes        uint64
	Disk_readBytes         uint64
	Disk_usedBytes         uint64
	Disk_totalBytes        uint64
	Cpu_physcal_core_count int
	Cpu_percent            float64
	Process_Cpu_percent    float64
	Cpu_logic_core_count   int
	Disk_IO_read_speed     uint64
	Disk_IO_write_speed    uint64
}

// 获取cpu相关信息
func getCpuInfo(i *Info) {
	totalPercent, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		log.Error("get Cpu_total_percent_ err", "err", err)
		return
	}
	i.Cpu_physcal_core_count, _ = cpu.Counts(false)
	i.Cpu_logic_core_count, _ = cpu.Counts(true)

	Cpu_percent_, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		log.Error("get Cpu_percent err", "err", err)
		return
	}

	i.Cpu_percent = Cpu_percent_[0]
	i.Cpu_total_percent = totalPercent[0]

	Cpu_, err := cpu.Percent(1*time.Second, true)
	var Cpu_total_percent float64
	for _, cp := range Cpu_ {
		Cpu_total_percent += cp
	}
	i.All_Cpu_total_percent = Cpu_total_percent / float64(i.Cpu_logic_core_count)

	pid := os.Getpid()
	proce, err := process.NewProcess(int32(pid))
	if err != nil {
		log.Error("get process err", "err", err)
		return
	}
	i.Process_Cpu_percent, err = proce.Percent(1 * time.Second)
	if err != nil {
		log.Error("get Process_Cpu_percent err", "err", err)
		return
	}
}

// 获取memory相关信息
func getMemInfo(i *Info) {
	info, err := mem.VirtualMemory()
	if err != nil {
		log.Error("get mem err", "err", err)
		return
	}
	i.Mem_totalSize = info.Total
	i.Mem_usedSize = info.Used
	i.Mem_usedPercent = info.UsedPercent
}

// 获取disk相关信息
func getDiskInfo(i *Info, path string) {
	mapStat, err := disk.IOCounters()
	if err != nil {
		log.Error("get disk info err", "err", err)
		return
	}
	i.Disk_readBytes = 0
	i.Disk_writeBytes = 0
	for _, stat := range mapStat {
		i.Disk_readBytes += stat.ReadBytes
		i.Disk_writeBytes += stat.WriteBytes
	}
	//开始采样。。。
	time.Sleep(1 * time.Second)
	var Disk_readBytes_ uint64 = 0
	var Disk_writeBytes_ uint64 = 0
	mapStat, err = disk.IOCounters()
	if err != nil {
		log.Error("get disk info err", "err", err)
		return
	}
	for _, stat := range mapStat {
		Disk_readBytes_ += stat.ReadBytes
		Disk_writeBytes_ += stat.WriteBytes
	}

	i.Disk_IO_read_speed = 2 * (Disk_readBytes_ - i.Disk_readBytes)
	i.Disk_IO_write_speed = 2 * (Disk_writeBytes_ - i.Disk_writeBytes)

	//磁盘使用情况
	info, _ := disk.Usage(path)
	i.Disk_totalBytes = info.Total
	i.Disk_usedBytes = info.Used
}

// 获取net相关信息
func getNetInfo(i *Info) {
	//获取net相关信息
	info, err := net.IOCounters(false)
	if err != nil {
		log.Error("get net info err", "err", err)
		return
	}
	for _, v := range info {
		i.Net_bytesRecv += v.BytesRecv
		i.Net_bytesSent += v.BytesSent
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
	i.Net_bytesRecv_speed = Net_bytesRecv_ - i.Net_bytesRecv
	i.Net_bytesSent_speed = Net_bytesSent_ - i.Net_bytesSent

}

// 获取Info中所有变量的值
func getInfo(path string) *Info {
	u := &Info{}
	// LINUX系统
	getCpuInfo(u)
	getMemInfo(u)
	getDiskInfo(u, path)
	getNetInfo(u)
	return u
}
