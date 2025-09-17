package kademlia

import (
	"reflect"
	"strings"
	"testing"
)

// TestNewKademliaID verifies that a KademliaID is correctly created from a valid hex string.
func TestNewKademliaID(t *testing.T) {
	// 1. Setup: Define a known hex string and its expected byte representation.
	hexString := "4142434445464748494a4b4c4d4e4f5051525354" // "ABCD..."
	expectedID := KademliaID{
		0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48,
		0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50,
		0x51, 0x52, 0x53, 0x54,
	}

	// 2. Action: Call the function we are testing.
	kademliaID := NewKademliaID(hexString)

	// 3. Assert: Check if the actual result matches the expected result.
	if !reflect.DeepEqual(*kademliaID, expectedID) {
		t.Errorf("NewKademliaID() failed. Got %v, expected %v", *kademliaID, expectedID)
	}
}

// TestNewRandomKademliaID checks if the function generates a non-nil ID and that
// two consecutive calls produce different IDs.
func TestNewRandomKademliaID(t *testing.T) {
	id1 := NewRandomKademliaID()
	id2 := NewRandomKademliaID()

	if id1 == nil {
		t.Fatal("NewRandomKademliaID() returned nil")
	}

	if id1.Equals(id2) {
		t.Errorf("NewRandomKademliaID() produced two identical IDs in a row, which is highly unlikely: %s", id1.String())
	}
}

// TestKademliaID_Less tests the comparison logic of two KademliaIDs.
func TestKademliaID_Less(t *testing.T) {
	id1 := NewKademliaID("000000000000000000000000000000000000000a") // Ends with 10
	id2 := NewKademliaID("000000000000000000000000000000000000000b") // Ends with 11
	id3 := NewKademliaID("000000000000000000000000000000000000000a") // Same as id1

	// Test case 1: id1 should be less than id2
	if !id1.Less(id2) {
		t.Errorf("Expected id1 to be less than id2, but it was not")
	}

	// Test case 2: id2 should NOT be less than id1
	if id2.Less(id1) {
		t.Errorf("Expected id2 not to be less than id1, but it was")
	}

	// Test case 3: id1 should NOT be less than id3 (they are equal)
	if id1.Less(id3) {
		t.Errorf("Expected equal IDs to return false for Less, but got true")
	}
}

// TestKademliaID_Equals tests the equality check between two KademliaIDs.
func TestKademliaID_Equals(t *testing.T) {
	id1 := NewKademliaID("ffffffffffffffffffffffffffffffffffffffff")
	id2 := NewKademliaID("ffffffffffffffffffffffffffffffffffffffff")
	id3 := NewKademliaID("0000000000000000000000000000000000000000")

	// Test case 1: Two identical IDs should be equal
	if !id1.Equals(id2) {
		t.Errorf("Expected id1 and id2 to be equal, but they were not")
	}

	// Test case 2: Two different IDs should not be equal
	if id1.Equals(id3) {
		t.Errorf("Expected id1 and id3 not to be equal, but they were")
	}
}

// TestKademliaID_CalcDistance verifies the XOR distance calculation.
func TestKademliaID_CalcDistance(t *testing.T) {
	// 1. Setup
	// ID1: 1111...1111
	id1 := NewKademliaID("ffffffffffffffffffffffffffffffffffffffff")
	// ID2: 0000...0000
	id2 := NewKademliaID("0000000000000000000000000000000000000000")
	// Expected distance: 1111...1111 XOR 0000...0000 = 1111...1111
	expectedDistance1 := id1

	// ID3: 1010...1010 (binary)
	id3 := NewKademliaID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	// ID4: 0101...0101 (binary)
	id4 := NewKademliaID("5555555555555555555555555555555555555555")
	// Expected distance: 1010... XOR 0101... = 1111...
	expectedDistance2 := id1

	// 2. Action and Assert
	distance1 := id1.CalcDistance(id2)
	if !distance1.Equals(expectedDistance1) {
		t.Errorf("CalcDistance failed for id1^id2. Got %s, expected %s", distance1.String(), expectedDistance1.String())
	}

	distance2 := id3.CalcDistance(id4)
	if !distance2.Equals(expectedDistance2) {
		t.Errorf("CalcDistance failed for id3^id4. Got %s, expected %s", distance2.String(), expectedDistance2.String())
	}

	// An ID's distance to itself should be 0
	distance3 := id1.CalcDistance(id1)
	if !distance3.Equals(id2) { // id2 is the zero ID
		t.Errorf("Distance to self should be 0. Got %s", distance3.String())
	}
}

// TestKademliaID_String verifies that the ID is correctly converted back to a hex string.
func TestKademliaID_String(t *testing.T) {
	hexStr := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	id := NewKademliaID(hexStr)
	resultStr := id.String()

	if !strings.EqualFold(hexStr, resultStr) {
		t.Errorf("String() conversion failed. Got '%s', expected '%s'", resultStr, hexStr)
	}
}
