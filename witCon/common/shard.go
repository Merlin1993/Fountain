package common

const (
	maxKnownBlocks = 102400

	tbQueueSize = 2000 //等待中的区块数量

	MaxExpectDiv = 2048
)

var (
	InitDirectShardCount = 8 // 分片数
)

// fnv取shard
func Shard(address Address) uint16 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(address); i++ {
		hash *= prime32
		hash ^= uint32(address[i])
	}
	return uint16(hash % uint32(InitDirectShardCount))
}
