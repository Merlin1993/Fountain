package rawbd

import (
	"sort"
	"strings"
	"sync"
	"witCon/common"
)

type MemDB struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDB() *MemDB {
	return &MemDB{
		db: make(map[string][]byte),
	}
}

func (db *MemDB) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return errMemoryDBClosed
	}
	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDB) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return false, errMemoryDBClosed
	}
	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDB) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return nil, errMemoryDBClosed
	}
	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errMemoryDBNotFound
}

func (db *MemDB) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return errMemoryDBClosed
	}
	delete(db.db, string(key))
	return nil
}

type iterator struct {
	inited bool
	keys   []string
	values [][]byte
}

func (it *iterator) Next() bool {
	// If the iterator was not yet initialized, do it now
	if !it.inited {
		it.inited = true
		return len(it.keys) > 0
	}
	// Iterator already initialize, advance it
	if len(it.keys) > 0 {
		it.keys = it.keys[1:]
		it.values = it.values[1:]
	}
	return len(it.keys) > 0
}

func (it *iterator) Error() error {
	return nil
}

func (it *iterator) Key() []byte {
	if len(it.keys) > 0 {
		return []byte(it.keys[0])
	}
	return nil
}

func (it *iterator) Value() []byte {
	if len(it.values) > 0 {
		return it.values[0]
	}
	return nil
}

func (it *iterator) Release() {
	it.keys, it.values = nil, nil
}

func (db *MemDB) NewIterator() Iterator {
	return db.NewIteratorWithStart(nil)
}

func (db *MemDB) NewIteratorWithStart(start []byte) Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var (
		st     = string(start)
		keys   = make([]string, 0, len(db.db))
		values = make([][]byte, 0, len(db.db))
	)
	// Collect the keys from the memory database corresponding to the given start
	for key := range db.db {
		if key >= st {
			keys = append(keys, key)
		}
	}
	// Sort the items and retrieve the associated valuesdatabase.go
	sort.Strings(keys)
	for _, key := range keys {
		values = append(values, db.db[key])
	}
	return &iterator{
		keys:   keys,
		values: values,
	}
}

func (db *MemDB) NewIteratorWithPrefix(prefix []byte) Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var (
		pr     = string(prefix)
		keys   = make([]string, 0, len(db.db))
		values = make([][]byte, 0, len(db.db))
	)
	// Collect the keys from the memory database corresponding to the given prefix
	for key := range db.db {
		if strings.HasPrefix(key, pr) {
			keys = append(keys, key)
		}
	}
	// Sort the items and retrieve the associated values
	sort.Strings(keys)
	for _, key := range keys {
		values = append(values, db.db[key])
	}
	return &iterator{
		keys:   keys,
		values: values,
	}
}

type keyValue struct {
	key    []byte
	value  []byte
	delete bool
}

type batch struct {
	db     *MemDB
	writes []keyValue
	size   int
}

func (db *MemDB) NewBatch() Batch {
	return &batch{
		db: db,
	}
}

func (b *batch) Put(key, value []byte) error {
	b.writes = append(b.writes, keyValue{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

func (b *batch) Delete(key []byte) error {
	b.writes = append(b.writes, keyValue{common.CopyBytes(key), nil, true})
	b.size += 1
	return nil
}

func (b *batch) ValueSize() int {
	return b.size
}

func (b *batch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, keyValue := range b.writes {
		if keyValue.delete {
			delete(b.db.db, string(keyValue.key))
			continue
		}
		b.db.db[string(keyValue.key)] = keyValue.value
	}
	return nil
}

func (b *batch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

func (b *batch) Replay(w KeyValueWriter) error {
	for _, keyValue := range b.writes {
		if keyValue.delete {
			if err := w.Delete(keyValue.key); err != nil {
				return err
			}
			continue
		}
		if err := w.Put(keyValue.key, keyValue.value); err != nil {
			return err
		}
	}
	return nil
}

func (db *MemDB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db = nil
	return nil
}
