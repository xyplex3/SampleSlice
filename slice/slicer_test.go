package slice_test

import (
	"slices"
	"testing"

	"sampleslice/detect"
	"sampleslice/slice"
)

// TestSliceAudio_NoSamples verifies that empty samples returns nil.
func TestSliceAudio_NoSamples(t *testing.T) {
	transients := []detect.Transient{
		{SamplePos: 100, Energy: 0.8},
	}
	config := slice.SliceConfig{
		PrePaddingMs:  10,
		PostPaddingMs: 20,
		RootNote:      "C3",
		Prefix:        "kick",
	}

	result := slice.SliceAudio(nil, transients, 44100, config)
	if result != nil {
		t.Errorf("expected nil for no samples, got %v", result)
	}
}

// TestSliceAudio_NoTransients verifies that no transients returns nil.
func TestSliceAudio_NoTransients(t *testing.T) {
	samples := make([]float64, 44100)
	config := slice.SliceConfig{
		PrePaddingMs:  10,
		PostPaddingMs: 20,
		RootNote:      "C3",
		Prefix:        "kick",
	}

	result := slice.SliceAudio(samples, nil, 44100, config)
	if result != nil {
		t.Errorf("expected nil for no transients, got %v", result)
	}
}

// TestSliceAudio_SingleTransient verifies slicing with a single transient.
func TestSliceAudio_SingleTransient(t *testing.T) {
	samples := make([]float64, 44100) // 1 second of silence
	transients := []detect.Transient{
		{SamplePos: 22050, Energy: 0.9}, // transient at 0.5s
	}
	config := slice.SliceConfig{
		PrePaddingMs:  10,
		PostPaddingMs: 20,
		RootNote:      "C3",
		Prefix:        "snare",
	}

	result := slice.SliceAudio(samples, transients, 44100, config)
	if len(result) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(result))
	}

	s := result[0]

	// Pre-padding: 10ms = 441 samples
	expectedStart := 22050 - 441
	if s.StartSample != expectedStart {
		t.Errorf("StartSample = %d, want %d", s.StartSample, expectedStart)
	}

	// Single transient should extend to end of audio
	if s.EndSample != 44100 {
		t.Errorf("EndSample = %d, want 44100", s.EndSample)
	}

	if s.Note != "C3" {
		t.Errorf("Note = %q, want %q", s.Note, "C3")
	}

	if s.Filename != "snare_C3.wav" {
		t.Errorf("Filename = %q, want %q", s.Filename, "snare_C3.wav")
	}
}

// TestSliceAudio_MultipleTransients verifies slicing with multiple transients.
func TestSliceAudio_MultipleTransients(t *testing.T) {
	samples := make([]float64, 44100)
	transients := []detect.Transient{
		{SamplePos: 10000, Energy: 0.8},
		{SamplePos: 20000, Energy: 0.9},
		{SamplePos: 30000, Energy: 0.7},
	}
	config := slice.SliceConfig{
		PrePaddingMs:  0, // no padding for easy math
		PostPaddingMs: 0,
		RootNote:      "D3",
		Prefix:        "hit",
	}

	result := slice.SliceAudio(samples, transients, 44100, config)
	if len(result) != 3 {
		t.Fatalf("expected 3 slices, got %d", len(result))
	}

	// First slice: start at transient 1, end at midpoint between 1 and 2
	if result[0].StartSample != 10000 {
		t.Errorf("Slice 0 StartSample = %d, want 10000", result[0].StartSample)
	}
	expectedMidpoint := (10000 + 20000) / 2 // 15000
	if result[0].EndSample != expectedMidpoint {
		t.Errorf("Slice 0 EndSample = %d, want %d", result[0].EndSample, expectedMidpoint)
	}
	if result[0].Note != "D3" {
		t.Errorf("Slice 0 Note = %q, want %q", result[0].Note, "D3")
	}

	// Second slice
	if result[1].StartSample != 20000 {
		t.Errorf("Slice 1 StartSample = %d, want 20000", result[1].StartSample)
	}
	expectedMidpoint2 := (20000 + 30000) / 2 // 25000
	if result[1].EndSample != expectedMidpoint2 {
		t.Errorf("Slice 1 EndSample = %d, want %d", result[1].EndSample, expectedMidpoint2)
	}
	if result[1].Note != "D#3" {
		t.Errorf("Slice 1 Note = %q, want %q", result[1].Note, "D#3")
	}

	// Third (last) slice should extend to end
	if result[2].EndSample != 44100 {
		t.Errorf("Slice 2 EndSample = %d, want 44100", result[2].EndSample)
	}
	if result[2].Note != "E3" {
		t.Errorf("Slice 2 Note = %q, want %q", result[2].Note, "E3")
	}
}

// TestSliceAudio_PrePaddingClamp verifies pre-padding doesn't go below 0.
func TestSliceAudio_PrePaddingClamp(t *testing.T) {
	samples := make([]float64, 44100)
	transients := []detect.Transient{
		{SamplePos: 100, Energy: 0.5}, // very early transient
	}
	config := slice.SliceConfig{
		PrePaddingMs:  100, // 100ms = 4410 samples, would go negative
		PostPaddingMs: 0,
		RootNote:      "C3",
		Prefix:        "test",
	}

	result := slice.SliceAudio(samples, transients, 44100, config)
	if len(result) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(result))
	}

	if result[0].StartSample != 0 {
		t.Errorf("StartSample = %d, want 0 (clamped)", result[0].StartSample)
	}
}

// TestSliceAudio_PostPaddingClamp verifies post-padding doesn't exceed buffer.
func TestSliceAudio_PostPaddingClamp(t *testing.T) {
	samples := make([]float64, 44100)
	transients := []detect.Transient{
		{SamplePos: 40000, Energy: 0.5},
		{SamplePos: 42000, Energy: 0.5},
	}
	config := slice.SliceConfig{
		PrePaddingMs:  0,
		PostPaddingMs: 10000, // huge post-padding
		RootNote:      "C3",
		Prefix:        "test",
	}

	result := slice.SliceAudio(samples, transients, 44100, config)
	if len(result) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(result))
	}

	// First slice midpoint = (40000+42000)/2 = 41000, + huge padding > 44100
	if result[0].EndSample != 44100 {
		t.Errorf("Slice 0 EndSample = %d, want 44100 (clamped)", result[0].EndSample)
	}
}

// TestSliceAudio_ChromaticAscension verifies notes ascend chromatically past B.
func TestSliceAudio_ChromaticAscension(t *testing.T) {
	samples := make([]float64, 441000) // 10 seconds
	// Create 13 transients to test octave rollover
	var transients []detect.Transient
	for i := 0; i < 13; i++ {
		transients = append(transients, detect.Transient{
			SamplePos: i*30000 + 1000,
			Energy:    0.8,
		})
	}

	config := slice.SliceConfig{
		PrePaddingMs:  0,
		PostPaddingMs: 0,
		RootNote:      "C3",
		Prefix:        "note",
	}

	result := slice.SliceAudio(samples, transients, 44100, config)

	// First note should be C3
	if result[0].Note != "C3" {
		t.Errorf("Note[0] = %q, want %q", result[0].Note, "C3")
	}
	// 13th note (index 12) should be C4 (one octave up)
	if result[12].Note != "C4" {
		t.Errorf("Note[12] = %q, want %q", result[12].Note, "C4")
	}
}

// TestExtractSamples_Normal verifies normal extraction.
func TestExtractSamples_Normal(t *testing.T) {
	samples := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	s := slice.AudioSlice{
		StartSample: 1,
		EndSample:   4,
	}

	result := slice.ExtractSamples(samples, s)
	expected := []float64{0.2, 0.3, 0.4}
	if !slices.Equal(result, expected) {
		t.Errorf("ExtractSamples() = %v, want %v", result, expected)
	}
}

// TestExtractSamples_StartBeyondBuffer verifies nil return when start is beyond buffer.
func TestExtractSamples_StartBeyondBuffer(t *testing.T) {
	samples := []float64{0.1, 0.2}
	s := slice.AudioSlice{
		StartSample: 10,
		EndSample:   20,
	}

	result := slice.ExtractSamples(samples, s)
	if result != nil {
		t.Errorf("expected nil for start beyond buffer, got %v", result)
	}
}

// TestExtractSamples_EndClamped verifies end is clamped to buffer length.
func TestExtractSamples_EndClamped(t *testing.T) {
	samples := []float64{0.1, 0.2, 0.3}
	s := slice.AudioSlice{
		StartSample: 1,
		EndSample:   100, // beyond buffer
	}

	result := slice.ExtractSamples(samples, s)
	expected := []float64{0.2, 0.3}
	if !slices.Equal(result, expected) {
		t.Errorf("ExtractSamples() = %v, want %v", result, expected)
	}
}

// TestNoteToMIDINote verifies MIDI note conversion.
func TestNoteToMIDINote(t *testing.T) {
	tests := []struct {
		note   string
		midiNote int
	}{
		{"C3", 48},
		{"C4", 60},
		{"A4", 69},
		{"C#3", 49},
		{"B2", 47},
	}

	for _, tt := range tests {
		result := slice.NoteToMIDINote(tt.note)
		if result != tt.midiNote {
			t.Errorf("NoteToMIDINote(%q) = %d, want %d", tt.note, result, tt.midiNote)
		}
	}
}

// TestNoteToMIDINote_A4 verifies A4 = MIDI 69 (standard tuning reference).
func TestNoteToMIDINote_A4(t *testing.T) {
	result := slice.NoteToMIDINote("A4")
	if result != 69 {
		t.Errorf("NoteToMIDINote(\"A4\") = %d, want 69", result)
	}
}

// TestSortBySamplePos verifies slices are sorted by StartSample ascending.
func TestSortBySamplePos(t *testing.T) {
	tests := []struct {
		name  string
		input []slice.AudioSlice
		want  []int
	}{
		{
			name: "already sorted",
			input: []slice.AudioSlice{
				{StartSample: 0},
				{StartSample: 100},
				{StartSample: 200},
			},
			want: []int{0, 100, 200},
		},
		{
			name: "reverse sorted",
			input: []slice.AudioSlice{
				{StartSample: 300},
				{StartSample: 100},
				{StartSample: 50},
			},
			want: []int{50, 100, 300},
		},
		{
			name:  "single element",
			input: []slice.AudioSlice{{StartSample: 42}},
			want:  []int{42},
		},
		{
			name:  "empty",
			input: []slice.AudioSlice{},
			want:  []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice.SortBySamplePos(tt.input)
			for i, s := range tt.input {
				if s.StartSample != tt.want[i] {
					t.Errorf("index %d: StartSample = %d, want %d", i, s.StartSample, tt.want[i])
				}
			}
		})
	}
}

// TestDeduplicateSlices verifies duplicate removal by audio content similarity.
func TestDeduplicateSlices(t *testing.T) {
	t.Run("threshold zero keeps all", func(t *testing.T) {
		// Even identical slices should be kept when threshold is 0 (disabled)
		samples := []float64{0.5, 0.5, 0.5, 0.5}
		slices := []slice.AudioSlice{
			{StartSample: 0, EndSample: 2},
			{StartSample: 2, EndSample: 4},
		}
		got := slice.DeduplicateSlices(slices, samples, 0)
		if len(got) != 2 {
			t.Errorf("threshold=0 got %d slices, want 2", len(got))
		}
	})

	t.Run("identical slices deduplicated at high threshold", func(t *testing.T) {
		// Two slices with identical constant content; cosine similarity = 1.0
		samples := make([]float64, 200)
		for i := range samples {
			samples[i] = 1.0
		}
		slices := []slice.AudioSlice{
			{StartSample: 0, EndSample: 100},
			{StartSample: 100, EndSample: 200},
		}
		got := slice.DeduplicateSlices(slices, samples, 0.99)
		if len(got) != 1 {
			t.Errorf("got %d slices, want 1 (duplicate removed)", len(got))
		}
	})

	t.Run("distinct slices kept", func(t *testing.T) {
		// First half full energy, second half silence → low similarity
		samples := make([]float64, 200)
		for i := 0; i < 100; i++ {
			samples[i] = 1.0
		}
		slices := []slice.AudioSlice{
			{StartSample: 0, EndSample: 100},
			{StartSample: 100, EndSample: 200},
		}
		got := slice.DeduplicateSlices(slices, samples, 0.99)
		if len(got) != 2 {
			t.Errorf("got %d slices, want 2 (distinct slices kept)", len(got))
		}
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		got := slice.DeduplicateSlices(nil, nil, 0.5)
		if got != nil {
			t.Errorf("expected nil for nil input, got %v", got)
		}
	})

	t.Run("first slice is always kept", func(t *testing.T) {
		samples := []float64{0.5, 0.5, 0.5, 0.5}
		slices := []slice.AudioSlice{
			{StartSample: 0, EndSample: 4},
		}
		got := slice.DeduplicateSlices(slices, samples, 0.9)
		if len(got) != 1 {
			t.Errorf("got %d slices, want 1", len(got))
		}
	})
}

// BenchmarkSliceAudio measures slicing performance.
func BenchmarkSliceAudio(b *testing.B) {
	samples := make([]float64, 441000) // 10 seconds
	var transients []detect.Transient
	for i := 0; i < 100; i++ {
		transients = append(transients, detect.Transient{
			SamplePos: i * 4000,
			Energy:    0.8,
		})
	}
	config := slice.SliceConfig{
		PrePaddingMs:  10,
		PostPaddingMs: 20,
		RootNote:      "C3",
		Prefix:        "bench",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = slice.SliceAudio(samples, transients, 44100, config)
	}
}

// BenchmarkExtractSamples measures sample extraction performance.
func BenchmarkExtractSamples(b *testing.B) {
	samples := make([]float64, 44100)
	s := slice.AudioSlice{
		StartSample: 1000,
		EndSample:   4000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = slice.ExtractSamples(samples, s)
	}
}