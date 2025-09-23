package storage

import (
	"sync"
	"time"
)

const ERR_INVALIDKEY string = "INVALID KEY"
const ERR_INVALIDVALUE string = "INVALID VALUE"
const ERR_INVALIDTIMESTAMP string = "INVALID TIMESTAMP"

type StoredInfo struct {
	information string
	timestamp   int64
}

type Storage struct {
	mutex   sync.Mutex
	hashmap map[string]*StoredInfo
	ttl     time.Duration
}

func NewStorageWithTTL(ttl time.Duration) *Storage {
	return &Storage{
		hashmap: make(map[string]*StoredInfo),
		ttl:     ttl,
	}
}
func NewStorage() *Storage {
	return &Storage{
		hashmap: make(map[string]*StoredInfo),
		ttl:     24 * time.Hour,
	}
}

func (storage *Storage) Get(key string) (string, bool) {
	if key == "" {
		panic(ERR_INVALIDKEY)
	}
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	value := storage.hashmap[key]
	info := ""
	if value != nil {
		value.timestamp = time.Now().UnixMilli()
		info = value.information
	}
	return info, value != nil
}

func (storage *Storage) Put(key string, value string) {
	storage.PutWithTimestamp(key, value, time.Now().UnixMilli())
}

func (storage *Storage) PutWithTimestamp(key string, value string, timestamp int64) {
	if key == "" {
		panic(ERR_INVALIDKEY)
	}
	if value == "" {
		panic(ERR_INVALIDVALUE)
	}
	if !storage.isTimestampValid(timestamp) {
		panic(ERR_INVALIDTIMESTAMP)
	}
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	storage.hashmap[key] = &StoredInfo{information: value, timestamp: timestamp}
}

func (storage *Storage) Size() int {
	return len(storage.hashmap)
}

func (storage *Storage) Clean() {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	for k, v := range storage.hashmap {
		if !storage.isTimestampValid(v.timestamp) {
			delete(storage.hashmap, k)
		}
	}
}

func (storage *Storage) isTimestampValid(timestamp int64) bool {
	return time.Now().UnixMilli()-timestamp <= storage.ttl.Milliseconds()
}
