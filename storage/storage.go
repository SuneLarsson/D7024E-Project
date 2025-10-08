package storage

import (
	"sync"
	"time"
)

const ERR_INVALIDKEY string = "INVALID KEY"
const ERR_INVALIDVALUE string = "INVALID VALUE"
const ERR_INVALIDTIMESTAMP string = "INVALID TIMESTAMP"

type StoredInfo struct {
	key                 string
	value               string
	timestamp           int64
	latestRepublishTime int64
	isOriginalPublisher bool
	next                *StoredInfo
}

type Storage struct {
	mutex       sync.Mutex
	information *StoredInfo
	ttl         time.Duration
}

//	func NewStorageWithTTL(ttl time.Duration) *Storage {
//		return &Storage{
//			hashmap: make(map[string]*StoredInfo),
//			ttl:     ttl,
//		}
//	}
func NewStorage(ttl time.Duration) *Storage {
	return &Storage{
		ttl:         ttl,
		information: nil}
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

func (storage *Storage) Put(key string, value string, isOriginal bool, orignalUploader bool) {
	storage.PutWithTimestamp(key, value, time.Now().UnixMilli(), isOriginal, orignalUploader)
}

func (storage *Storage) PutWithTimestamp(key string, value string, timestamp int64, isOriginal bool, orignalUploader bool) {
	if key == "" {
		panic(ERR_INVALIDKEY)
	}
	if value == "" {
		panic(ERR_INVALIDVALUE)
	}
	if !storage.isTimestampValid(timestamp) {
		panic(ERR_INVALIDTIMESTAMP)
	}
	storage.iterativePut(key, value, timestamp, isOriginal, orignalUploader)
}

func (storage *Storage) iterativePut(key string, value string, timestamp int64, isOriginal bool, orignalUploader bool) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	var previousInfo *StoredInfo = nil
	currentInfo := storage.information
	information := &StoredInfo{
		key:                 key,
		value:               value,
		isOriginalPublisher: isOriginal,
		timestamp:           timestamp,
	}
	if orignalUploader {
		information.latestRepublishTime = timestamp
	}
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
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
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
		if !storage.isTimestampValid(currentInfo.timestamp) {
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

func (storage *Storage) isTimestampValid(timestamp int64) bool {
	age := time.Now().UnixMilli() - timestamp
	return age <= storage.ttl.Milliseconds()
}

func (storage *Storage) GetValueAndMetadataForReplication(key string) (value string, isOriginal bool, found bool) {
	if key == "" {
		panic(ERR_INVALIDKEY)
	}

	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	currentInfo := storage.information
	for currentInfo != nil {
		if currentInfo.key == key {
			if !storage.isTimestampValid(currentInfo.latestRepublishTime) {
				return "", false, false
			}
			// Refresh the timestamp on access.
			currentInfo.timestamp = time.Now().UnixMilli()
			return currentInfo.value, currentInfo.isOriginalPublisher, true
		}
		if currentInfo.key > key {
			break
		}
		currentInfo = currentInfo.next
	}
	return "", false, false
}
