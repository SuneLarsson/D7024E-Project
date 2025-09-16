package storage

import (
	"testing"
	"time"
)

// Test what happens when trying to use the Get function with an empty string as parameter
func TestGetEmptyKey(t *testing.T) {
	storage := NewStorage()
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDKEY {
			t.Error("When the key is empty, ERR_INVALIDKEY should be thrown")
		}
	}()
	storage.Get("")
}

// Test what happens when trying to use the Put function with an empty string as key parameter
func TestPutEmptyKey(t *testing.T) {
	storage := NewStorage()
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDKEY {
			t.Error("When the key is empty, ERR_INVALIDKEY should be thrown")
		}
	}()
	storage.Put("", "")
}

// Test what happens when trying to use the Put function with an empty string as value parameter
func TestPutEmptyValue(t *testing.T) {
	storage := NewStorage()
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDVALUE {
			t.Error("When the value is empty ERR_INVALIDVALUE should be thrown")
		}
	}()
	storage.Put("0", "")
}

// Test what happens when trying to use the PutWithTimestamp function with an empty string as key parameter
func TestPutWithTimestampEmptyKey(t *testing.T) {
	storage := NewStorage()
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDKEY {
			t.Error("When the key is empty, ERR_INVALIDKEY should be thrown")
		}
	}()
	storage.PutWithTimestamp("", "", 0)
}

// Test what happens when trying to use the PutWithTimestamp function with an empty string as value parameter
func TestPutWithTimestampEmptyValue(t *testing.T) {
	storage := NewStorage()
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDVALUE {
			t.Error("When the value is empty ERR_INVALIDVALUE should be thrown")
		}
	}()
	storage.PutWithTimestamp("0", "", 0)
}

// Test what happens when trying to use the PutWithTimestamp function with an timestamp from more than a day ago
func TestPutWithTimestampInvalidTimeStamp(t *testing.T) {
	storage := NewStorage()
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDTIMESTAMP {
			t.Error("When the value is empty ERR_INVALIDTIMESTAMP should be thrown")
		}
	}()
	storage.PutWithTimestamp("key", "value", 0)
}

// Test for good behaviour
func TestGoodBehaviour(t *testing.T) {
	storage := NewStorage()
	key := "thisismykey"
	value := "thisismyvalue"
	storage.Put(key, value)
	valueStored := storage.Get(key)
	if value != valueStored {
		t.Error("Value expected:", value, "and value received:", valueStored)
	}
}

// Test for Get unknown key
func TestGetUnknownKey(t *testing.T) {
	defer func() {
		if err := recover(); err == nil || err != ERR_UNKNOWNKEY {
			t.Error("When the key does not exist in storage, ERR_UNKNOWNKEY should be thrown")
		}
	}()
	storage := NewStorage()
	key := "keywithnoknownvalue"
	storage.Get(key)
}

// Test for Put already existing key
func TestPutExistingKey(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error("No error should be thrown when assigning a new value to an existing key")
		}
	}()
	storage := NewStorage()
	key := "thisismykey"
	value := "thisismyFIRSTvalue"
	storage.Put(key, value)
	value = "thisismySECONDvalue"
	storage.Put(key, value)
	valueStored := storage.Get(key)
	if value != valueStored {
		t.Error("Value expected:", value, "and value received:", valueStored)
	}
}

// Test Size is 0 upon creation
func TestSizeCreation(t *testing.T) {
	storage := NewStorage()
	sizeStorage := storage.Size()
	if sizeStorage != 0 {
		t.Error("Storage is supposed to be empty on creation, size found to be", sizeStorage)
	}
}

// Test that size grows up when adding a new element
func TestSizeGrowing(t *testing.T) {
	storage := NewStorage()
	storage.Put("key", "value")
	sizeStorage := storage.Size()
	if sizeStorage != 1 {
		t.Error("Storage size value expected: 1. Size found is", sizeStorage)
	}
}

// Test of cleaning ancient values
func TestCleaning(t *testing.T) {
	storage := NewStorage()
	timestamp := time.Now().AddDate(0, 0, -1).Add(100 * time.Millisecond).UnixMilli()
	storage.PutWithTimestamp("key", "value", timestamp)
	time.Sleep(200 * time.Millisecond)
	storage.Clean()
	sizeStorage2 := storage.Size()
	if sizeStorage2 != 0 {
		t.Error("Error in cleaning of ancient content")
	}
}

// Test of reset of timestamp ancient values
func TestResetTimestampBeforeCleaning(t *testing.T) {
	storage := NewStorage()
	timestamp := time.Now().AddDate(0, 0, -1).Add(100 * time.Millisecond).UnixMilli()
	storage.PutWithTimestamp("key", "value", timestamp)
	sizeStorage1 := storage.Size()
	time.Sleep(110 * time.Millisecond)
	storage.Get("key")
	storage.Clean()
	sizeStorage2 := storage.Size()
	if sizeStorage1 != sizeStorage2 {
		t.Error("Error in reseting timestamp of ancient content")
	}
}
