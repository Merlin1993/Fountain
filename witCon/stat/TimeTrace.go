package stat

import (
	"sort"
	"strconv"
	"sync"
	"time"
	"witCon/common"

	lru "github.com/hashicorp/golang-lru"
)

const mapSize = 10000

var (
	HashTimeTrace   = newTimeTrace()
	DBlockTimeTrace = newTimeTrace()
)

type Trace struct {
	count     int64
	countTime int64
	curTime   int64
}

type TimeTrace struct {
	tbTrace *lru.ARCCache
	stat    sync.Map
	finTime *FinStat
	finLock sync.RWMutex
	lock2   sync.RWMutex
	count   int
	sum     int64
}

type FinStat struct {
	start   *lru.ARCCache
	avgTime int64
	count   int64
}

type TimeStat struct {
	avgTime int64
	curTime int64
	count   int64
	name    string
}

type TimeStats []*TimeStat

func (ts TimeStats) Len() int {
	return len(ts)
}

func (ts TimeStats) Less(i, j int) bool {
	return ts[i].avgTime > ts[j].avgTime
}

func (ts TimeStats) Swap(i, j int) {
	tmp := ts[i]
	ts[i] = ts[j]
	ts[j] = tmp
}

func newTimeTrace() *TimeTrace {
	tt := &TimeTrace{}
	tt.tbTrace, _ = lru.NewARC(mapSize)
	tt.finLock = sync.RWMutex{}
	st, _ := lru.NewARC(1000000)
	tt.finTime = &FinStat{
		start:   st,
		avgTime: 0,
		count:   0,
	}
	return tt
}

func (t *TimeTrace) AddLen(size int) {
	t.lock2.Lock()
	defer t.lock2.Unlock()
	t.sum += int64(size)
	t.count++
}

func (t *TimeTrace) AddDBTime(num uint64, cn int, key string) {
	if cn <= 0 {
		return
	}
	nlt := time.Now().UnixMicro()
	if v, ok := t.tbTrace.Get(num); ok {
		lt := v.(int64)
		ts, ok := t.stat.Load(key)
		var tsv *Trace
		if !ok {
			tsv = &Trace{
				count:     0,
				countTime: 0,
				curTime:   0,
			}
		} else {
			tsv = ts.(*Trace)
		}
		tsv.countTime += nlt - lt
		tsv.count++
		tsv.curTime = nlt - lt
		t.stat.Store(key, tsv)
	}
	t.tbTrace.Add(num, nlt)
}

func (t *TimeTrace) AddTBTime(hash common.Hash, key string) {
	nlt := time.Now().UnixMicro()
	if v, ok := t.tbTrace.Get(hash); ok {
		lt := v.(int64)
		ts, ok := t.stat.Load(key)
		var tsv *Trace
		if !ok {
			tsv = &Trace{
				count:     0,
				countTime: 0,
			}
		} else {
			tsv = ts.(*Trace)
		}
		tsv.countTime += nlt - lt
		tsv.count++
		t.stat.Store(key, tsv)
	}
	t.tbTrace.Add(hash, nlt)
}

func (t *TimeTrace) NewT(hash common.Hash) {
	t.finTime.start.Add(hash, time.Now().UnixMicro())
}

func (t *TimeTrace) FinNum(num uint64) {
	now := time.Now().UnixMicro()
	re, ok := t.finTime.start.Get(num)
	if !ok {
		return
	}
	ls := re.(int64)
	if now > ls {
		consume := now - ls
		t.finTime.avgTime = (t.finTime.avgTime*t.finTime.count + consume) / (t.finTime.count + 1)
		t.finTime.count++
	}
}

func (t *TimeTrace) FinT(hash common.Hash) {
	now := time.Now().UnixMicro()
	re, ok := t.finTime.start.Get(hash)
	if !ok {
		return
	}
	ls := re.(int64)
	if now > ls {
		consume := now - ls
		t.finTime.avgTime = (t.finTime.avgTime*t.finTime.count + consume) / (t.finTime.count + 1)
		t.finTime.count++
	}
}

func (t *TimeTrace) EndDBTime(num uint64, cn int) {
	if cn <= 0 {
		return
	}
	nlt := time.Now().UnixMicro()
	if v, ok := t.tbTrace.Get(num); ok {
		lt := v.(int64)
		ts, ok := t.stat.Load("end")
		var tsv *Trace
		if !ok {
			tsv = &Trace{
				count:     0,
				countTime: 0,
			}
		} else {
			tsv = ts.(*Trace)
		}
		tsv.countTime += nlt - lt
		tsv.count++
		t.stat.Store("end", tsv)
		t.tbTrace.Remove(num)
	}
	t.FinNum(num)
}

func (t *TimeTrace) EndTBTime(hash common.Hash) {
	nlt := time.Now().UnixMicro()
	if v, ok := t.tbTrace.Get(hash); ok {
		lt := v.(int64)
		ts, ok := t.stat.Load("end")
		var tsv *Trace
		if !ok {
			tsv = &Trace{
				count:     0,
				countTime: 0,
			}
		} else {
			tsv = ts.(*Trace)
		}
		tsv.countTime += nlt - lt
		tsv.count++
		t.stat.Store("end", tsv)
		t.tbTrace.Remove(hash)
	}
	t.FinT(hash)
}

func (t *TimeTrace) StartTBKeyTime(key string) {
	nlt := time.Now().UnixMicro()
	hash := common.BytesToHash([]byte("keyS" + key))
	t.tbTrace.Add(hash, nlt)
}

func (t *TimeTrace) EndTBKeyTime(key string) {
	nlt := time.Now().UnixMicro()
	hash := common.BytesToHash([]byte("keyS" + key))
	if v, ok := t.tbTrace.Get(hash); ok {
		lt := v.(int64)
		ts, ok := t.stat.Load(key)
		var tsv *Trace
		if !ok {
			tsv = &Trace{
				count:     0,
				countTime: 0,
			}
		} else {
			tsv = ts.(*Trace)
		}
		tsv.countTime += nlt - lt
		tsv.count++
		t.stat.Store(key, tsv)
		t.tbTrace.Remove(hash)
	}
}

func (t *TimeTrace) StatTB() (string, string) {
	result := ""
	ts := make(TimeStats, 0, 20)
	t.stat.Range(func(key, value interface{}) bool {
		v := value.(*Trace)
		k := key.(string)
		avgt := v.countTime / v.count
		ts = append(ts, &TimeStat{
			avgTime: avgt,
			curTime: v.curTime,
			count:   v.count,
			name:    k,
		})
		return true
	})
	sort.Sort(ts)
	for _, v := range ts {
		result += v.name + ",avg:" + strconv.FormatInt(v.avgTime, 10) + ",cur:" + strconv.FormatInt(v.curTime, 10) + "," + strconv.FormatInt(v.count, 10)
		result += ";"
	}
	t.lock2.Lock()
	defer t.lock2.Unlock()

	result2 := strconv.FormatInt(t.finTime.avgTime, 10) + "," + strconv.FormatInt(t.finTime.count, 10) + "," + strconv.FormatInt(t.sum/int64(t.count+1), 10)
	t.sum = 0
	t.count = 0

	return result, result2
}
