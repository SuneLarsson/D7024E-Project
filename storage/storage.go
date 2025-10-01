package storage

import (
	"sync"
	"time"
)

const ERR_INVALIDKEY string = "INVALID KEY"
const ERR_INVALIDVALUE string = "INVALID VALUE"
const ERR_INVALIDTIMESTAMP string = "INVALID TIMESTAMP"

type StoredInfo struct {
	key       string
	value     string
	timestamp int64
	next      *StoredInfo
}

type Storage struct {
	mutex       sync.Mutex
	information *StoredInfo
}

func NewStorage() *Storage {
	return &Storage{information: nil}
}

func (storage *Storage) Get(key string) (string, bool) {
	if key == "" {
		panic(ERR_INVALIDKEY)
	}
	return storage.iterativeGet(key)
}

func (storage *Storage) iterativeGet(key string) (string, bool) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	currentInfo := storage.information
	returnedInfo := ""
	found := false
	for {
		if currentInfo == nil {
			break
		}
		if currentInfo.key == key {
			currentInfo.timestamp = time.Now().UnixMilli()
			returnedInfo = currentInfo.value
			found = true
			break
		}
		currentInfo = currentInfo.next
	}
	return returnedInfo, found
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
	storage.iterativePut(key, value, timestamp)
}

func (storage *Storage) iterativePut(key string, value string, timestamp int64) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	var previousInfo *StoredInfo = nil
	currentInfo := storage.information
	information := &StoredInfo{key: key, value: value, timestamp: timestamp}
	for {
		keepGoing := true
		if currentInfo == nil {
			keepGoing = false
		} else {
			if currentInfo.key == key {
				keepGoing = false
				information.next = currentInfo.next
			}
			if currentInfo.key > key {
				keepGoing = false
				information.next = currentInfo
			}
		}

		if !keepGoing {
			if previousInfo == nil {
				storage.information = information
			} else {
				previousInfo.next = information
			}
			break
		}
		previousInfo = currentInfo
		currentInfo = currentInfo.next
	}
}

func (storage *Storage) Size() int {
	length := 0
	currentInfo := storage.information
	for currentInfo != nil {
		length++
		currentInfo = currentInfo.next
	}
	return length
}

func (storage *Storage) GetKeys() []string {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	returnArray := []string{}
	currentInfo := storage.information
	for currentInfo != nil {
		returnArray = append(returnArray, currentInfo.key)
		currentInfo = currentInfo.next
	}
	return returnArray
}

func (storage *Storage) Clean() {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	var previousInfo *StoredInfo = nil
	currentInfo := storage.information
	for currentInfo != nil {
		if !isTimestampValid(currentInfo.timestamp) {
			if previousInfo == nil {
				storage.information = currentInfo.next
			} else {
				previousInfo.next = currentInfo.next
			}
			currentInfo = currentInfo.next
			continue
		}
		previousInfo = currentInfo
		currentInfo = currentInfo.next
	}
}

func isTimestampValid(timestamp int64) bool {
	return time.Now().UnixMilli()-timestamp <= 86400000
}
