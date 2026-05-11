// Package krz tests for hash generation and extraction utilities.
package krz

import (
	"testing"
)

// TestGenerateHash verifies hash construction from object type and ID.
func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name       string
		id         uint16
		objectType uint16
		want       uint16
	}{
		{"keymap id 1", 1, T_KEYMAP, 0x8201},
		{"keymap id 0", 0, T_KEYMAP, 0x8200},
		{"keymap id 255", 255, T_KEYMAP, 0x82FF},
		{"program id 1", 1, T_PROGRAM, 0x8401},
		{"program id 2", 2, T_PROGRAM, 0x8402},
		{"effect id 1", 1, T_EFFECT, 0x7101},
		{"id truncated to low byte", 256, T_KEYMAP, 0x8200}, // 256 & 0xFF = 0
		{"id 1000 low byte 0xE8", 1000, T_KEYMAP, 0x82E8},
		{"setup type", 1, T_SETUP, 0x8701},
		{"zero id and type", 0, 0, 0x0000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateHash(tt.id, tt.objectType)
			if got != tt.want {
				t.Errorf("GenerateHash(%d, 0x%02x) = 0x%04x, want 0x%04x",
					tt.id, tt.objectType, got, tt.want)
			}
		})
	}
}

// TestTypeFromHash verifies extraction of the object type byte from a hash.
func TestTypeFromHash(t *testing.T) {
	tests := []struct {
		name string
		hash uint16
		want uint16
	}{
		{"keymap type", 0x8201, 0x82},
		{"program type", 0x8401, 0x84},
		{"effect type", 0x7101, 0x71},
		{"song type", 0x7000, 0x70},
		{"zero hash", 0x0000, 0x00},
		{"max hash", 0xFFFF, 0xFF},
		{"setup type", 0x8701, 0x87},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TypeFromHash(tt.hash)
			if got != tt.want {
				t.Errorf("TypeFromHash(0x%04x) = 0x%02x, want 0x%02x", tt.hash, got, tt.want)
			}
		})
	}
}

// TestIDFromHash verifies extraction of the low-byte ID from a hash.
func TestIDFromHash(t *testing.T) {
	tests := []struct {
		name string
		hash uint16
		want uint16
	}{
		{"id 1", 0x8201, 0x01},
		{"id 2", 0x8402, 0x02},
		{"id 0xFF", 0x82FF, 0xFF},
		{"id 0", 0x8200, 0x00},
		{"zero hash", 0x0000, 0x00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IDFromHash(tt.hash)
			if got != tt.want {
				t.Errorf("IDFromHash(0x%04x) = 0x%02x, want 0x%02x", tt.hash, got, tt.want)
			}
		})
	}
}

// TestGenerateHash_TypeInHighByte verifies that GenerateHash stores the
// object type in the high byte so TypeFromHash can recover it.
func TestGenerateHash_TypeInHighByte(t *testing.T) {
	types := []uint16{T_PROGRAM, T_KEYMAP, T_EFFECT, T_SONG, T_SETUP}
	for _, objType := range types {
		hash := GenerateHash(1, objType)
		got := TypeFromHash(hash)
		if got != objType&0xFF {
			t.Errorf("type 0x%02x: TypeFromHash(GenerateHash) = 0x%02x, want 0x%02x",
				objType, got, objType&0xFF)
		}
	}
}

// BenchmarkGenerateHash measures hash generation performance.
func BenchmarkGenerateHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateHash(uint16(i), T_KEYMAP)
	}
}
