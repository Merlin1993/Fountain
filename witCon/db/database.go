package rawbd

import "io"

const IdealBatchSize = 100 * 1024

type KeyValueReader interface {
	Has(key []byte) (bool, error)

	Get(key []byte) ([]byte, error)
}

type Reader interface {
	KeyValueReader
}

type KeyValueWriter interface {
	Put(key []byte, value []byte) error

	Delete(key []byte) error
}

type Writer interface {
	KeyValueWriter
}

type Batch interface {
	KeyValueWriter

	ValueSize() int

	Write() error

	Reset()

	Replay(w KeyValueWriter) error
}

type Batcher interface {
	NewBatch() Batch
}

type Iterator interface {
	Next() bool

	Error() error

	Key() []byte

	Value() []byte

	Release()
}

type Iteratee interface {
	NewIterator() Iterator
	NewIteratorWithStart(start []byte) Iterator
	NewIteratorWithPrefix(prefix []byte) Iterator
}

type Database interface {
	Reader
	Writer
	Batcher
	Iteratee
	io.Closer
}
