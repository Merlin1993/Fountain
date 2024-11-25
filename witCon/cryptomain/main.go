package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/shirou/gopsutil/cpu"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"witCon/crypto"
)

func main() {
	x()
}

func x() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	count := atomic.Int32{}
	count.Store(0)

	sk, _ := crypto.GenerateKey()
	hash := crypto.HashSum([]byte{0x01, 0x02})
	result, _ := crypto.Sign(hash[:], sk)
	pks := crypto.SchnorrPk(crypto.CompressPubKey(&sk.PublicKey))

	baseConcurrency := 28
	concurrentIncrement := 4
	verifySigPool, _ := ants.NewPool(28)
	exchangeGoPool, _ := ants.NewPool(5)
	fmt.Println(fmt.Sprintf("verify sig pool : %v", 28))
	tj := 14
	ti := 500
	go func() {
		for {
			exchangeGoPool.Submit(func() {
				var wg sync.WaitGroup
				for j := 0; j < tj; j++ {
					wg.Add(1)
					verifySigPool.Submit(func() {
						for i := 0; i < ti; i++ {
							tpk, err := crypto.SigToPub(hash[:], result)
							tpks := crypto.CompressPubKey(tpk)
							if err != nil || !bytes.Equal(tpks, pks[:]) {
								fmt.Println("pk fail", "err", err, "pk", hex.EncodeToString(pks[:]), "tx", hex.EncodeToString(tpks))
								return
							}
							count.Add(1)
						}
						wg.Done()
					})
				}
				wg.Wait()
			})
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second) // 每隔30秒执行
			baseConcurrency += concurrentIncrement
			// 调整 ants 池的大小
			verifySigPool.Tune(baseConcurrency)
			ti *= 2
			fmt.Println(fmt.Sprintf("Increased verify sig pool : %v", ti))
		}
	}()

	var last int32
	timer := time.NewTimer(0 * time.Second)
	//<-timer.C
	for {
		select {
		case <-timer.C:
			tl := last
			last = count.Load()
			fmt.Println("timerLoop", "count", last-tl)
			percent, err := cpu.Percent(time.Second, false)
			if err == nil {
				fmt.Printf("CPU Usage: %.2f%%\n", percent[0])
			}
			timer.Reset(1 * time.Second)
		}
	}
}
