package storage

import (
	"sync"
	"time"
)

const ERR_INVALIDKEY string = "INVALID KEY"
const ERR_INVALIDVALUE string = "INVALID VALUE"
const ERR_UNKNOWNKEY string = "UNKNOWN KEY"
const ERR_INVALIDTIMESTAMP string = "INVALID TIMESTAMP"

type StoredInfo struct {
	information string
	timestamp   int64
}

type Storage struct {
	mutex   sync.Mutex
	hashmap map[string]*StoredInfo
}

func NewStorage() *Storage {
	return &Storage{hashmap: make(map[string]*StoredInfo)}
}

func (storage *Storage) Get(key string) string {
	if key == "" {
		panic(ERR_INVALIDKEY)
	}
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	value := storage.hashmap[key]
	if value == nil {
		panic(ERR_UNKNOWNKEY)
	}
	value.timestamp = time.Now().UnixMilli()
	return value.information
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
	if !isTimestampValid(timestamp) {
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
		if !isTimestampValid(v.timestamp) {
			delete(storage.hashmap, k)
		}
	}
}

func isTimestampValid(timestamp int64) bool {
	return time.Now().UnixMilli()-timestamp <= 86400000
}
