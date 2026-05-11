// Package krz tests for Keymap: construction, sample addition,
// and binary serialization covering all format variants.
package krz

import (
	"encoding/binary"
	"testing"
)

// -----------------------------------------------------------------------
// Tests for NewKeymap
// -----------------------------------------------------------------------

func TestNewKeymap(t *testing.T) {
	tests := []struct {
		name      string
		id        uint16
		nameStr   string
		wantID    uint16
		wantName  string
		wantHash  uint16
	}{
			{
			name:     "normal name",
			id:       1,
			nameStr:  "Kick",
			wantID:   1,
			wantName: "Kick",
			wantHash: 0x8201,
			},
			{
			name:     "long name truncated",
			id:       2,
			nameStr:  "ThisIsAVeryLongName",
			wantID:   2,
			wantName: "ThisIsAVeryLongN",
			wantHash: 0x8202,
			},
			{
			name:     "empty name",
			id:       3,
			nameStr:  "",
			wantID:   3,
			wantName: "",
			wantHash: 0x8203,
			},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			km := NewKeymap(tc.id, tc.nameStr)
			if km.ID != tc.wantID {
				t.Errorf("ID = %d, want %d", km.ID, tc.wantID)
			}
			if km.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", km.Name, tc.wantName)
			}
			if km.Hash != tc.wantHash {
				t.Errorf("Hash = 0x%04x, want 0x%04x", km.Hash, tc.wantHash)
			}
			if len(km.Samples) != 0 {
				t.Error("Samples should be empty initially")
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for Keymap.AddSample
// -----------------------------------------------------------------------

func TestKeymap_AddSample(t *testing.T) {
	km := NewKeymap(1, "Test")

	// First sample
	km.AddSample([]int16{100, 200}, 36, 36, 36, 8, false)
	if len(km.Samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(km.Samples))
	}

	s := km.Samples[0]
	if s.RootNote != 36 {
		t.Errorf("RootNote = %d, want 36", s.RootNote)
	}
	if s.Format != 8 {
		t.Errorf("Format = %d, want 8", s.Format)
	}
	if len(s.RawData) == 0 {
		t.Error("RawData should not be empty")
	}

	// Second sample
	km.AddSample([]int16{50}, 60, 60, 60, 3, true)
	if len(km.Samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(km.Samples))
	}

	s2 := km.Samples[1]
	if s2.Format != 3 {
		t.Errorf("Format = %d, want 3", s2.Format)
	}
	if !s2.Compressed {
		t.Error("Compressed should be true")
	}
}

// -----------------------------------------------------------------------
// Tests for Keymap.Serialize
// -----------------------------------------------------------------------

func TestKeymap_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Keymap
		check   func(*testing.T, []byte)
	}{
			{
			name: "single uncompressed 8-bit sample",
			setup: func() *Keymap {
				km := NewKeymap(1, "Kick")
				km.AddSample([]int16{128, 256, 0, -128}, 36, 36, 36, 8, false)
				return km
				},
			check: func(t *testing.T, data []byte) {
				// Verify header bytes: hash 0x8201
				if binary.BigEndian.Uint16(data[0:2]) != 0x8201 {
					t.Errorf("header hash = 0x%04x, want 0x8201", binary.BigEndian.Uint16(data[0:2]))
				}
				// sample count should be 1
				if binary.BigEndian.Uint16(data[2:4]) != 1 {
					t.Errorf("sample count = %d, want 1", binary.BigEndian.Uint16(data[2:4]))
				}
				// name length should be 4 ("Kick")
				if int(data[10]) != 4 {
					t.Errorf("name length = %d, want 4", data[10])
				}
				},
			},
			{
			name: "single adpcm sample",
			setup: func() *Keymap {
				km := NewKeymap(1, "ADPCM")
				km.AddSample([]int16{100, 200, 150}, 60, 60, 60, 3, true)
				return km
				},
			check: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("data is empty")
				}
				},
			},
			{
			name: "single 16-bit signed sample",
			setup: func() *Keymap {
				km := NewKeymap(1, "Bit16")
				km.AddSample([]int16{32767, -32768, 0}, 48, 48, 48, 16, false)
				return km
				},
			check: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("data is empty")
				}
				},
			},
			{
			name: "multiple samples",
			setup: func() *Keymap {
				km := NewKeymap(2, "Multi")
				km.AddSample([]int16{100}, 36, 36, 36, 8, false)
				km.AddSample([]int16{200}, 42, 42, 42, 8, false)
				km.AddSample([]int16{50}, 48, 48, 48, 8, false)
				return km
				},
			check: func(t *testing.T, data []byte) {
				// hash should be 0x8202
				if binary.BigEndian.Uint16(data[0:2]) != 0x8202 {
					t.Errorf("header hash = 0x%04x, want 0x8202", binary.BigEndian.Uint16(data[0:2]))
				}
				// sample count should be 3
				if binary.BigEndian.Uint16(data[2:4]) != 3 {
					t.Errorf("sample count = %d, want 3", binary.BigEndian.Uint16(data[2:4]))
				}
				},
			},
			{
			name: "empty keymap no samples",
			setup: func() *Keymap {
				return NewKeymap(5, "Empty")
				},
			check: func(t *testing.T, data []byte) {
				if binary.BigEndian.Uint16(data[2:4]) != 0 {
					t.Errorf("sample count = %d, want 0", binary.BigEndian.Uint16(data[2:4]))
				}
				},
			},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			km := tc.setup()
			data, err := km.Serialize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, data)
		})
	}
}

// -----------------------------------------------------------------------
// Tests for Keymap serialization name padding
// -----------------------------------------------------------------------

func TestKeymap_SerializeNamePadding(t *testing.T) {
	tests := []struct {
		nameStr  string
		wantLen  int // expected name length byte value
	}{
			{"", 0},
			{"A", 1},
				{"Exact16Chars!!", 14},
			{"TooLongNameForField", 16}, // truncated
	}

	for _, tc := range tests {
		t.Run(tc.nameStr, func(t *testing.T) {
			km := NewKeymap(1, tc.nameStr)
			data, err := km.Serialize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			nameLen := int(data[10])
			if nameLen != tc.wantLen {
				t.Errorf("name length = %d, want %d", nameLen, tc.wantLen)
			}
			// Name data starts at offset 11
			if nameLen > 0 {
				actualName := string(data[11 : 11+nameLen])
				expectedName := tc.nameStr
				if len(expectedName) > 16 {
					expectedName = expectedName[:16]
				}
				if actualName != expectedName {
					t.Errorf("name = %q, want %q", actualName, expectedName)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for 16-bit to 8-bit conversion
// -----------------------------------------------------------------------

func TestKeymap_SixteenBitToEightBitConversion(t *testing.T) {
	// Test that 16-bit values are properly converted to 8-bit
	km := NewKeymap(1, "Conv")
	// Use values that span the 16-bit range
	samples := []int16{0, 128, 255, 256, 32767, -1, -128, -32768}
	km.AddSample(samples, 60, 60, 60, 16, false)

	data, err := km.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("serialized data is empty")
	}
}

// -----------------------------------------------------------------------
// Tests for Keymap sample data integrity
// -----------------------------------------------------------------------

func TestKeymap_SampleDataIntegrity(t *testing.T) {
	originalSamples := []int16{100, 200, 50, 150, 250}
	km := NewKeymap(1, "Integrity")
	km.AddSample(originalSamples, 60, 60, 60, 8, false)

	// RawData is stored as []byte (converted from int16), verify non-empty
	if len(km.Samples[0].RawData) == 0 {
		t.Error("sample data should not be empty")
	}
}

// -----------------------------------------------------------------------
// Tests for Keymap Hash computation
// -----------------------------------------------------------------------

func TestKeymap_HashComputation(t *testing.T) {
	tests := []struct {
		id       uint16
		wantHash uint16
	}{
			{1, 0x8201},
			{0, 0x8200},
			{255, 0x82FF},
			{1000, 0x82E8}, // 1000 & 0xFF = 232 = 0xE8
	}

	for _, tc := range tests {
		km := NewKeymap(tc.id, "Test")
		if km.Hash != tc.wantHash {
			t.Errorf("id %d: hash = 0x%04x, want 0x%04x", tc.id, km.Hash, tc.wantHash)
		}
	}
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------
// Tests for Keymap.CalculateSize
// -----------------------------------------------------------------------

func TestKeymap_CalculateSize(t *testing.T) {
	t.Run("empty keymap returns even size", func(t *testing.T) {
		km := NewKeymap(1, "Test")
		size, err := km.CalculateSize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if size <= 0 {
			t.Errorf("size = %d, want > 0", size)
		}
		if size%2 != 0 {
			t.Errorf("size = %d, want even (2-byte padded)", size)
		}
	})

	t.Run("size with samples is even", func(t *testing.T) {
		km := NewKeymap(2, "WithSample")
		km.AddSample([]int16{100, 200, 300}, 48, 48, 48, 8, false)
		size, err := km.CalculateSize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if size%2 != 0 {
			t.Errorf("size = %d, want even (2-byte padded)", size)
		}
	})

	t.Run("calculated size is at least serialized length", func(t *testing.T) {
		km := NewKeymap(3, "Check")
		km.AddSample([]int16{1, 2, 3, 4, 5}, 36, 36, 36, 8, false)
		data, err := km.Serialize()
		if err != nil {
			t.Fatalf("serialize error: %v", err)
		}
		size, err := km.CalculateSize()
		if err != nil {
			t.Fatalf("calculate size error: %v", err)
		}
		if size < len(data) {
			t.Errorf("CalculateSize() = %d, less than serialized length %d", size, len(data))
		}
	})

	t.Run("padding produces even size from any input", func(t *testing.T) {
		for _, numSamples := range []int{1, 2, 3, 7, 11} {
			samples := make([]int16, numSamples)
			for i := range samples {
				samples[i] = int16(i * 100)
			}
			km := NewKeymap(uint16(numSamples), "P")
			km.AddSample(samples, 60, 60, 60, 8, false)
			size, err := km.CalculateSize()
			if err != nil {
				t.Fatalf("numSamples=%d: unexpected error: %v", numSamples, err)
			}
			if size%2 != 0 {
				t.Errorf("numSamples=%d: size = %d, want even", numSamples, size)
			}
		}
	})
}

// -----------------------------------------------------------------------
// Round-trip: serialize and verify structure
// -----------------------------------------------------------------------

func TestKeymap_SerializeRoundTrip(t *testing.T) {
	km := NewKeymap(3, "Round")
	km.AddSample([]int16{10, 20, 30, 40, 50}, 36, 36, 36, 8, false)
	km.AddSample([]int16{60, 70, 80}, 42, 42, 42, 3, true)

	data, err := km.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's non-empty and starts with correct hash
	if len(data) < 4 {
		t.Fatal("data too short")
	}
	if binary.BigEndian.Uint16(data[0:2]) != 0x8203 {
		t.Errorf("hash = 0x%04x, want 0x8203", binary.BigEndian.Uint16(data[0:2]))
	}
	if binary.BigEndian.Uint16(data[2:4]) != 2 {
		t.Errorf("sample count = %d, want 2", binary.BigEndian.Uint16(data[2:4]))
	}
}