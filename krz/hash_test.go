// Package krz tests for hash generation and extraction utilities.
package krz

import "testing"

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name       string
		id         uint16
		objectType uint16
		want       uint16
	}{
		{"program id 200", 200, T_PROGRAM, 0x90c8},
		{"keymap id 200", 200, T_KEYMAP, 0x94c8},
		{"sample id 201", 201, T_SAMPLE, 0x98c9},
		{"program id 1", 1, T_PROGRAM, 0x9001},
		{"zero id and type", 0, 0, 0x0000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateHash(tt.id, tt.objectType)
			if got != tt.want {
				t.Errorf("GenerateHash(%d, %d) = 0x%04x, want 0x%04x", tt.id, tt.objectType, got, tt.want)
			}
		})
	}
}

func TestTypeFromHash(t *testing.T) {
	tests := []struct {
		name string
		hash uint16
		want uint16
	}{
		{"program", 0x90c8, T_PROGRAM},
		{"keymap", 0x94c8, T_KEYMAP},
		{"sample", 0x98c9, T_SAMPLE},
		{"zero hash", 0x0000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TypeFromHash(tt.hash); got != tt.want {
				t.Errorf("TypeFromHash(0x%04x) = %d, want %d", tt.hash, got, tt.want)
			}
		})
	}
}

func TestIDFromHash(t *testing.T) {
	tests := []struct {
		name string
		hash uint16
		want uint16
	}{
		{"program id 200", 0x90c8, 200},
		{"keymap id 200", 0x94c8, 200},
		{"sample id 201", 0x98c9, 201},
		{"zero hash", 0x0000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IDFromHash(tt.hash); got != tt.want {
				t.Errorf("IDFromHash(0x%04x) = %d, want %d", tt.hash, got, tt.want)
			}
		})
	}
}

// TestGenerateHash_RoundTrip verifies TypeFromHash/IDFromHash recover the
// original type and ID for every object type this package writes.
func TestGenerateHash_RoundTrip(t *testing.T) {
	types := []uint16{T_PROGRAM, T_KEYMAP, T_SAMPLE}
	for _, objType := range types {
		for _, id := range []uint16{0, 1, 200, 1023} {
			hash := GenerateHash(id, objType)
			if got := TypeFromHash(hash); got != objType {
				t.Errorf("type %d id %d: TypeFromHash(GenerateHash) = %d, want %d", objType, id, got, objType)
			}
			if got := IDFromHash(hash); got != id {
				t.Errorf("type %d id %d: IDFromHash(GenerateHash) = %d, want %d", objType, id, got, id)
			}
		}
	}
}

// BenchmarkGenerateHash measures hash generation performance.
func BenchmarkGenerateHash(b *testing.B) {
	for i := 0; b.Loop(); i++ {
		_ = GenerateHash(uint16(i), T_KEYMAP)
	}
}
