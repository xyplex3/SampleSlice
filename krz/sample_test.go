package krz

import (
	"encoding/binary"
	"testing"
)

func decodeSamplePayload(t *testing.T, obj []byte) []byte {
	t.Helper()
	if len(obj) < 26 {
		t.Fatalf("object too short: %d bytes", len(obj))
	}
	return obj[26:]
}

func TestSample_Serialize_CommonHeader(t *testing.T) {
	s := &Sample{ID: 201, Name: "kick001", RootNote: 36, SampleRate: 44100, NumSamples: 3}
	obj := s.Serialize()

	wantHash := GenerateHash(201, T_SAMPLE)
	if got := binary.BigEndian.Uint16(obj[4:6]); got != wantHash {
		t.Errorf("hash = 0x%04x, want 0x%04x", got, wantHash)
	}
	blocksize := int32(binary.BigEndian.Uint32(obj[0:4]))
	if int(-blocksize) != len(obj) {
		t.Errorf("blocksize = %d, want -%d", blocksize, len(obj))
	}
	name := obj[10:26]
	if string(name) != "kick001         " {
		t.Errorf("name = %q, want %q", name, "kick001         ")
	}
}

func TestSample_Serialize_Fields(t *testing.T) {
	s := &Sample{ID: 201, Name: "kick001", RootNote: 36, SampleRate: 44100, NumSamples: 19968, StartWord: 0}
	payload := decodeSamplePayload(t, s.Serialize())

	if len(payload) != 70 {
		t.Fatalf("payload length = %d, want 70", len(payload))
	}

	if rootkey := payload[14]; rootkey != 36 {
		t.Errorf("rootkey = %d, want 36", rootkey)
	}
	if flags := payload[15]; flags != 0xF0 {
		t.Errorf("flags = 0x%02x, want 0xF0 (one-shot)", flags)
	}

	// maxPitch: confirmed formula, round(100*root + 1200*log2(48000/sr)).
	// root=36, sr=44100 -> 3747 (matches a real sample at this exact root/rate).
	maxPitch := int16(binary.BigEndian.Uint16(payload[18:20]))
	if maxPitch != 3747 {
		t.Errorf("maxPitch = %d, want 3747", maxPitch)
	}

	sampleStart := binary.BigEndian.Uint32(payload[22:26])
	sampleEnd := binary.BigEndian.Uint32(payload[34:38])
	if sampleStart != 0 {
		t.Errorf("sampleStart = %d, want 0", sampleStart)
	}
	if sampleEnd != 19967 {
		t.Errorf("sampleEnd = %d, want 19967 (StartWord + NumSamples - 1)", sampleEnd)
	}

	samplePeriod := binary.BigEndian.Uint32(payload[42:46])
	if samplePeriod != 22675 {
		t.Errorf("samplePeriod = %d, want 22675 (1e9/44100 truncated)", samplePeriod)
	}

	wantEnv := []int16{-1, 1, 0, 0, -1600, 0}
	for copyIdx := 0; copyIdx < 2; copyIdx++ {
		base := 46 + copyIdx*12
		for i, want := range wantEnv {
			got := int16(binary.BigEndian.Uint16(payload[base+i*2 : base+i*2+2]))
			if got != want {
				t.Errorf("envelope copy %d value %d = %d, want %d", copyIdx, i, got, want)
			}
		}
	}
}

func TestSample_Serialize_StartWordOffset(t *testing.T) {
	// Second sample in a bank starts right after the first, matching real
	// files (kick002's sampleStart == kick001's sampleEnd + 1).
	s := &Sample{ID: 202, Name: "kick002", RootNote: 37, SampleRate: 44100, NumSamples: 10832, StartWord: 19968}
	payload := decodeSamplePayload(t, s.Serialize())

	sampleStart := binary.BigEndian.Uint32(payload[22:26])
	sampleEnd := binary.BigEndian.Uint32(payload[34:38])
	loopStart := binary.BigEndian.Uint32(payload[30:34])
	if sampleStart != 19968 {
		t.Errorf("sampleStart = %d, want 19968", sampleStart)
	}
	if sampleEnd != 30799 {
		t.Errorf("sampleEnd = %d, want 30799", sampleEnd)
	}
	if loopStart != sampleStart {
		t.Errorf("loopStart = %d, want %d (== sampleStart for one-shot, matching real files)", loopStart, sampleStart)
	}
}
