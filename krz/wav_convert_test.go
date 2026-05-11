// Package krz tests for WAV-to-KRZ conversion functions including
// 8-bit unsigned conversion, 16-bit big-endian conversion, and ADPCM compression.
package krz

import (
	"testing"
)

// -----------------------------------------------------------------------
// Tests for ConvertTo8BitUnsigned
// -----------------------------------------------------------------------

func TestConvertTo8BitUnsigned(t *testing.T) {
	tests := []struct {
		name   string
		input  []int16
		expect []uint8
	}{
		{
			name:   "silence (0 maps to 128)",
			input:  []int16{0},
			expect: []uint8{128},
		},
		{
			name:   "minimum 16-bit value maps to 0",
			input:  []int16{-32768},
			expect: []uint8{0},
		},
		{
			name:   "maximum 16-bit value maps to 255",
			input:  []int16{32767},
			expect: []uint8{255},
		},
		{
			name:   "mid range values",
			input:  []int16{-16384, 0, 16383},
			expect: []uint8{64, 128, 191},
		},
		{
			name:   "empty input",
			input:  []int16{},
			expect: []uint8{},
		},
		{
			name:   "multiple samples",
			input:  []int16{-32768, -16384, 0, 16383, 32767},
			expect: []uint8{0, 64, 128, 191, 255},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertTo8BitUnsigned(tc.input)
			if len(result) != len(tc.expect) {
				t.Fatalf("length = %d, want %d", len(result), len(tc.expect))
			}
			for i, v := range tc.expect {
				if result[i] != v {
					t.Errorf("result[%d] = %d, want %d", i, result[i], v)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for ConvertWAVToKRZFormat
// -----------------------------------------------------------------------

func TestConvertWAVToKRZFormat_16Bit(t *testing.T) {
	tests := []struct {
		name   string
		input  []int16
		check  func(*testing.T, []byte)
	}{
		{
			name:  "single zero sample becomes 0x00 0x00",
			input: []int16{0},
			check: func(t *testing.T, data []byte) {
				if len(data) != 2 {
					t.Fatalf("length = %d, want 2", len(data))
				}
				if data[0] != 0 || data[1] != 0 {
					t.Errorf("expected 0x00 0x00, got 0x%02x 0x%02x", data[0], data[1])
				}
			},
		},
		{
			name:  "maximum positive value 32767 = 0x7FFF",
			input: []int16{32767},
			check: func(t *testing.T, data []byte) {
				if data[0] != 0x7F || data[1] != 0xFF {
					t.Errorf("expected 0x7F 0xFF, got 0x%02x 0x%02x", data[0], data[1])
				}
			},
		},
		{
			name:  "minimum negative value -32768 = 0x8000",
			input: []int16{-32768},
			check: func(t *testing.T, data []byte) {
				if data[0] != 0x80 || data[1] != 0x00 {
					t.Errorf("expected 0x80 0x00, got 0x%02x 0x%02x", data[0], data[1])
				}
			},
		},
		{
			name:  "empty input produces empty output",
			input: []int16{},
			check: func(t *testing.T, data []byte) {
				if len(data) != 0 {
					t.Errorf("expected empty output, got %d bytes", len(data))
				}
			},
		},
		{
			name:  "multiple samples produce correct length",
			input: []int16{100, -200, 300},
			check: func(t *testing.T, data []byte) {
				if len(data) != 6 {
					t.Errorf("length = %d, want 6", len(data))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := ConvertWAVToKRZFormat(tc.input, 16)
			tc.check(t, data)
		})
	}
}

func TestConvertWAVToKRZFormat_8Bit(t *testing.T) {
	tests := []struct {
		name   string
		input  []int16
		check  func(*testing.T, []byte)
	}{
		{
			name:  "zero sample becomes 128",
			input: []int16{0},
			check: func(t *testing.T, data []byte) {
				if len(data) != 1 || data[0] != 128 {
					t.Errorf("expected [128], got %v", data)
				}
			},
		},
		{
			name:  "full range samples",
			input: []int16{-32768, 32767},
			check: func(t *testing.T, data []byte) {
				if len(data) != 2 {
					t.Fatalf("length = %d, want 2", len(data))
				}
				if data[0] != 0 {
					t.Errorf("data[0] = %d, want 0", data[0])
				}
				if data[1] != 255 {
					t.Errorf("data[1] = %d, want 255", data[1])
				}
			},
		},
		{
			name:  "empty input",
			input: []int16{},
			check: func(t *testing.T, data []byte) {
				if len(data) != 0 {
					t.Errorf("expected empty output, got %d bytes", len(data))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := ConvertWAVToKRZFormat(tc.input, 0)
			tc.check(t, data)
		})
	}
}

// -----------------------------------------------------------------------
// Tests for CompressADPCM
// -----------------------------------------------------------------------

func TestCompressADPCM(t *testing.T) {
	tests := []struct {
		name      string
		input     []uint8
		checkLen  int
		checkData func(*testing.T, []byte)
	}{
		{
			name:      "empty input returns nil",
			input:     []uint8{},
			checkLen:  0,
			checkData: func(t *testing.T, data []byte) {
				if data != nil {
					t.Error("expected nil for empty input")
				}
			},
		},
		{
			name:      "single sample produces one output byte",
			input:     []uint8{128},
			checkLen:  1,
			checkData: func(t *testing.T, data []byte) {
				// High nibble should contain the delta for the first sample
				// Low nibble should be 0 since there's no second sample
				lowNibble := data[0] & 0x0F
				if lowNibble != 0 {
					t.Errorf("low nibble = %d, want 0 (no second sample)", lowNibble)
				}
			},
		},
		{
			name:      "two samples produce one output byte",
			input:     []uint8{128, 128},
			checkLen:  1,
			checkData: func(t *testing.T, data []byte) {
				// Both samples are the same constant value, deltas should be small
				t.Logf("output byte: 0x%02x", data[0])
			},
		},
		{
			name:      "four samples produce two output bytes",
			input:     []uint8{128, 130, 126, 128},
			checkLen:  2,
			checkData: func(t *testing.T, data []byte) {
				t.Logf("output bytes: 0x%02x 0x%02x", data[0], data[1])
			},
		},
		{
			name:      "odd number of samples pads with zero in low nibble",
			input:     []uint8{128, 130, 126},
			checkLen:  2,
			checkData: func(t *testing.T, data []byte) {
				// Third sample goes into high nibble of second byte, low nibble is 0
				lowNibble := data[1] & 0x0F
				if lowNibble != 0 {
					t.Errorf("low nibble of second byte = %d, want 0", lowNibble)
				}
			},
		},
		{
			name:      "constant signal produces small deltas",
			input:     []uint8{100, 100, 100, 100, 100, 100},
			checkLen:  3,
			checkData: func(t *testing.T, data []byte) {
				// After the first sample, all deltas should be 0 or very small
				t.Logf("output bytes: %v", data)
			},
		},
		{
			name:      "output length formula matches expectation",
			input:     makeConstantSignal(20, 150),
			checkLen:  10, // (20 + 1) / 2 = 10
			checkData: func(t *testing.T, data []byte) {
				// No specific assertion, just verify length
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CompressADPCM(tc.input)
			if tc.checkLen == 0 {
				if result != nil {
					t.Errorf("expected nil, got %d bytes", len(result))
				}
				return
			}
			if len(result) != tc.checkLen {
				t.Errorf("length = %d, want %d", len(result), tc.checkLen)
				return
			}
			tc.checkData(t, result)
		})
	}
}

// makeConstantSignal creates a slice of n samples all set to val
func makeConstantSignal(n int, val uint8) []uint8 {
	s := make([]uint8, n)
	for i := range s {
		s[i] = val
	}
	return s
}

// -----------------------------------------------------------------------
// Tests for encodeDelta
// -----------------------------------------------------------------------

func TestEncodeDelta(t *testing.T) {
	tests := []struct {
		name      string
		actual    uint8
		predicted int32
		step      uint8
		check     func(*testing.T, int32)
	}{
		{
			name:      "exact match produces zero delta",
			actual:    128,
			predicted: 128,
			step:      7,
			check: func(t *testing.T, delta int32) {
				if delta != 0 {
					t.Errorf("delta = %d, want 0", delta)
				}
			},
		},
		{
			name:      "delta is bounded to -15..15",
			actual:    255,
			predicted: 0,
			step:      7,
			check: func(t *testing.T, delta int32) {
				if delta < -15 || delta > 15 {
					t.Errorf("delta = %d, want range [-15, 15]", delta)
				}
			},
		},
		{
			name:      "small positive difference",
			actual:    135,
			predicted: 128,
			step:      7,
			check: func(t *testing.T, delta int32) {
				if delta < 0 {
					t.Errorf("expected positive delta, got %d", delta)
				}
			},
		},
		{
			name:      "small negative difference",
			actual:    120,
			predicted: 128,
			step:      7,
			check: func(t *testing.T, delta int32) {
				if delta >= 0 {
					t.Errorf("expected negative delta, got %d", delta)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			delta := encodeDelta(tc.actual, tc.predicted, tc.step)
			tc.check(t, delta)
		})
	}
}

// -----------------------------------------------------------------------
// Tests for updatePrediction
// -----------------------------------------------------------------------

func TestUpdatePrediction(t *testing.T) {
	tests := []struct {
		name      string
		predicted int32
		delta     int32
		step      uint8
		check     func(*testing.T, int32)
	}{
		{
			name:      "zero delta keeps prediction",
			predicted: 128,
			delta:     0,
			step:      7,
			check: func(t *testing.T, result int32) {
				if result != 128 {
					t.Errorf("result = %d, want 128", result)
				}
			},
		},
		{
			name:      "clamped to minimum 0",
			predicted: 0,
			delta:     -15, // signedDelta = -15
			step:      10,
			check: func(t *testing.T, result int32) {
				if result < 0 {
					t.Errorf("result = %d, want >= 0", result)
				}
			},
		},
		{
			name:      "clamped to maximum 255",
			predicted: 255,
			delta:     15, // signedDelta = -1 (15 > 7 so 15-16=-1)
			step:      10,
			check: func(t *testing.T, result int32) {
				if result > 255 {
					t.Errorf("result = %d, want <= 255", result)
				}
			},
		},
		{
			name:      "positive delta increases prediction",
			predicted: 100,
			delta:     2, // signedDelta = 2
			step:      7,
			check: func(t *testing.T, result int32) {
				if result <= 100 {
					t.Errorf("result = %d, want > 100", result)
				}
			},
		},
		{
			name:      "negative delta decreases prediction",
			predicted: 100,
			delta:     -2, // signedDelta = -2
			step:      7,
			check: func(t *testing.T, result int32) {
				if result >= 100 {
					t.Errorf("result = %d, want < 100", result)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updatePrediction(tc.predicted, tc.delta, tc.step)
			tc.check(t, result)
		})
	}
}

// -----------------------------------------------------------------------
// Tests for updateStepIndex
// -----------------------------------------------------------------------

func TestUpdateStepIndex(t *testing.T) {
	indexAdjust := [16]int{
		0, 0, 0, 0, 1, 1, 1, 1,
		2, 2, 2, 2, 3, 3, 3, 3,
	}

	tests := []struct {
		name         string
		currentIndex int
		delta        int32
		expectMin    int
		expectMax    int
	}{
		{
			name:         "zero delta keeps index",
			currentIndex: 5,
			delta:        0,
			expectMin:    5,
			expectMax:    5,
		},
		{
			name:         "clamped to minimum 0",
			currentIndex: 0,
			delta:        -3, // adjust = 0, so stays 0
			expectMin:    0,
			expectMax:    0,
		},
		{
			name:         "clamped to maximum 15",
			currentIndex: 15,
			delta:        15, // adjust = 3, but clamped to 15
			expectMin:    15,
			expectMax:    15,
		},
		{
			name:         "index increases with large delta",
			currentIndex: 5,
			delta:        14, // adjust = 3
			expectMin:    8,
			expectMax:    8,
		},
		{
			name:         "small delta does not change index",
			currentIndex: 10,
			delta:        2, // adjust = 0
			expectMin:    10,
			expectMax:    10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateStepIndex(tc.currentIndex, tc.delta, indexAdjust)
			if result < tc.expectMin || result > tc.expectMax {
				t.Errorf("result = %d, want [%d, %d]", result, tc.expectMin, tc.expectMax)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Integration: ConvertTo8BitUnsigned then CompressADPCM
// -----------------------------------------------------------------------

func TestConvertThenCompressADPCM(t *testing.T) {
	samples := []int16{0, 100, -100, 32767, -32768, 16383, -16384}
	eightBit := ConvertTo8BitUnsigned(samples)
	compressed := CompressADPCM(eightBit)

	if len(compressed) == 0 {
		t.Error("compressed output is empty")
	}

	expectedLen := (len(eightBit) + 1) / 2
	if len(compressed) != expectedLen {
		t.Errorf("compressed length = %d, want %d", len(compressed), expectedLen)
	}
}