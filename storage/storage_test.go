package storage

import (
	"testing"
	"time"
)

const Test_ttl = 24 * time.Hour
const tExpire = 25 // in hours

// Test what happens when trying to use the Get function with an empty string as parameter
func TestGetEmptyKey(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDKEY {
			t.Error("When the key is empty, ERR_INVALIDKEY should be thrown")
		}
	}()
	storage.Get("")
}

// Test what happens when trying to use the Put function with an empty string as key parameter
func TestPutEmptyKey(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDKEY {
			t.Error("When the key is empty, ERR_INVALIDKEY should be thrown")
		}
	}()
	storage.Put("", "", false, false)
}

// Test what happens when trying to use the Put function with an empty string as value parameter
func TestPutEmptyValue(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDVALUE {
			t.Error("When the value is empty ERR_INVALIDVALUE should be thrown")
		}
	}()
	storage.Put("0", "", false, false)
}

// Test what happens when trying to use the PutWithTimestamp function with an empty string as key parameter
func TestPutWithTimestampEmptyKey(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDKEY {
			t.Error("When the key is empty, ERR_INVALIDKEY should be thrown")
		}
	}()
	storage.PutWithTimestamp("", "", 0, false, false)
}

// Test what happens when trying to use the PutWithTimestamp function with an empty string as value parameter
func TestPutWithTimestampEmptyValue(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDVALUE {
			t.Error("When the value is empty ERR_INVALIDVALUE should be thrown")
		}
	}()
	storage.PutWithTimestamp("0", "", 0, false, false)
}

// Test what happens when trying to use the PutWithTimestamp function with an timestamp from more than a day ago
func TestPutWithTimestampInvalidTimeStamp(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	defer func() {
		if err := recover(); err == nil || err != ERR_INVALIDTIMESTAMP {
			t.Error("When the value is empty ERR_INVALIDTIMESTAMP should be thrown")
		}
	}()
	storage.PutWithTimestamp("key", "value", 0, false, false)
}

// Test for good behaviour
func TestGoodBehaviour(t *testing.T) {

	storage := NewStorage(Test_ttl, tExpire)
	key := "thisismykey"
	value := "thisismyvalue"
	storage.Put(key, value, true, true)
	var valueStored string
	valueStored, _ = storage.Get(key)
	if value != valueStored {
		t.Error("Value expected:", value, "and value received:", valueStored)
	}
}

// Test for Get unknown key
func TestGetUnknownKey(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	key := "keywithnoknownvalue"
	var exists bool
	_, exists = storage.Get(key)
	if exists {
		t.Error("Unknown key should not return existing value")
	}
}

// Test for Get when last value in order
func TestGetLastValue(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	storage.Put("aaa", "wrongValue", false, false)
	storage.Put("bbb", "wrongValue", false, false)
	storage.Put("ccc", "rightValue", true, true)
	storedValue, _ := storage.Get("ccc")
	if storedValue != "rightValue" {
		t.Error("Last value was not correctly recuperated")
	}
}

// Test for Put already existing key
func TestPutExistingKey(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Error("No error should be thrown when assigning a new value to an existing key")
		}
	}()
	storage := NewStorage(Test_ttl, tExpire)
	key := "thisismykey"
	value := "thisismyFIRSTvalue"
	storage.Put(key, value, true, true)
	value = "thisismySECONDvalue"
	storage.Put(key, value, true, true)
	var valueStored string
	valueStored, _ = storage.Get(key)
	if value != valueStored {
		t.Error("Value expected:", value, "and value received:", valueStored)
	}
}

// Test Size is 0 upon creation
func TestSizeCreation(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	sizeStorage := storage.Size()
	if sizeStorage != 0 {
		t.Error("Storage is supposed to be empty on creation, size found to be", sizeStorage)
	}
}

// Test that size grows up when adding a new element
func TestSizeGrowing(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	storage.Put("key", "value", true, true)
	sizeStorage := storage.Size()
	if sizeStorage != 1 {
		t.Error("Storage size value expected: 1. Size found is", sizeStorage)
	}
}

// Test of cleaning ancient values
func TestCleaning(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	timestamp := time.Now().AddDate(0, 0, -1).Add(100 * time.Millisecond).UnixMilli()
	storage.PutWithTimestamp("key", "value", timestamp, true, true)
	time.Sleep(200 * time.Millisecond)
	storage.Clean()
	sizeStorage2 := storage.Size()
	if sizeStorage2 != 0 {
		t.Error("Error in cleaning of ancient content")
	}
}

// Test of cleaning with a recent value before
func TestCleaningWithRecentThenAncient(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	storage.Put("aaa", "value", true, true)
	timestamp := time.Now().AddDate(0, 0, -1).Add(100 * time.Millisecond).UnixMilli()
	storage.PutWithTimestamp("bbb", "value", timestamp, true, true)
	time.Sleep(200 * time.Millisecond)
	storage.Clean()
	sizeStorage := storage.Size()
	if sizeStorage != 1 {
		t.Error("Error in cleaning of ancient content")
	}
}

// Test of reset of timestamp ancient values
func TestResetTimestampBeforeCleaning(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)
	timestamp := time.Now().AddDate(0, 0, -1).Add(100 * time.Millisecond).UnixMilli()
	storage.PutWithTimestamp("key", "value", timestamp, true, true)
	sizeStorage1 := storage.Size()
	time.Sleep(110 * time.Millisecond)
	storage.Get("key")
	storage.Clean()
	sizeStorage2 := storage.Size()
	if sizeStorage1 != sizeStorage2 {
		t.Error("Error in reseting timestamp of ancient content")
	}
}

// Test the recuperation of the keys of the storage
func TestGetKeys(t *testing.T) {
	storage := NewStorage(Test_ttl, tExpire)

	if len(storage.GetKeys()) != 0 {
		t.Error("Storage should be empty on creation")
	}

	storage.Put("bbb", "value", true, true)

	testKeyArray([]string{"bbb"}, storage.GetKeys(), t)

	storage.Put("aaa", "value", true, true)

	testKeyArray([]string{"aaa", "bbb"}, storage.GetKeys(), t)

	storage.Put("ccc", "value", true, true)

	testKeyArray([]string{"aaa", "bbb", "ccc"}, storage.GetKeys(), t)

	storage.Put("aaa", "value", true, true)

	testKeyArray([]string{"aaa", "bbb", "ccc"}, storage.GetKeys(), t)

	storage.Put("bbb", "value", true, true)

	testKeyArray([]string{"aaa", "bbb", "ccc"}, storage.GetKeys(), t)

	storage.Put("ccc", "value", true, true)

	testKeyArray([]string{"aaa", "bbb", "ccc"}, storage.GetKeys(), t)

}

func testKeyArray(reference []string, keyArray []string, t *testing.T) {
	if len(reference) != len(keyArray) {
		t.Errorf("There should be %d %s", len(reference), "keys")
	}

	for index := range keyArray {
		if reference[index] != keyArray[index] {
			t.Errorf("There is a difference between %s and %s", toString(reference), toString(keyArray))
		}
	}
}

func toString(anArray []string) string {
	returnValue := "["
	for i := range anArray {
		returnValue += anArray[i] + ","
	}
	returnValue += "]"
	return returnValue
}
