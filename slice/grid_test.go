package slice

import (
	"testing"
)

const testSampleRate = uint32(44100)

// samplesPerSliceAt returns the expected slice length for the given parameters.
func samplesPerSliceAt(bpm float64, beatsPerBar, loopBars int) int {
	samplesPerBeat := (float64(testSampleRate) * 60.0) / bpm
	samplesPerBar := samplesPerBeat * float64(beatsPerBar)
	return max(int(samplesPerBar*float64(loopBars)+0.5), 1)
}

func TestSliceByGrid_CorrectSliceLength(t *testing.T) {
	// 120 BPM, 4/4, 1 bar: samplesPerBeat=22050, samplesPerBar=88200
	spb := samplesPerSliceAt(120, 4, 1)
	if spb != 88200 {
		t.Fatalf("expected 88200 samples/bar, got %d", spb)
	}

	samples := make([]float64, spb*4)
	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	for i, s := range slices {
		got := s.EndSample - s.StartSample
		if got != spb {
			t.Errorf("slice %d: expected length %d, got %d", i, spb, got)
		}
	}
}

func TestSliceByGrid_ExactFourBars(t *testing.T) {
	spb := samplesPerSliceAt(120, 4, 1)
	samples := make([]float64, spb*4)

	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	if len(slices) != 4 {
		t.Fatalf("expected 4 slices, got %d", len(slices))
	}
	for i, s := range slices {
		wantStart := i * spb
		if s.StartSample != wantStart {
			t.Errorf("slice %d: StartSample=%d, want %d", i, s.StartSample, wantStart)
		}
		if s.EndSample != wantStart+spb {
			t.Errorf("slice %d: EndSample=%d, want %d", i, s.EndSample, wantStart+spb)
		}
	}
}

func TestSliceByGrid_LoopBarsDouble(t *testing.T) {
	spb1 := samplesPerSliceAt(120, 4, 1)
	spb2 := samplesPerSliceAt(120, 4, 2)
	samples := make([]float64, spb1*4)

	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    2,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	if len(slices) != 2 {
		t.Fatalf("expected 2 slices with loop-bars=2, got %d", len(slices))
	}
	for i, s := range slices {
		got := s.EndSample - s.StartSample
		if got != spb2 {
			t.Errorf("slice %d: expected length %d, got %d", i, spb2, got)
		}
	}
}

func TestSliceByGrid_WaltzBeatsPerBar(t *testing.T) {
	// 90 BPM, 3/4: samplesPerBeat=44100*60/90=29400, samplesPerBar=29400*3=88200
	spb := samplesPerSliceAt(90, 3, 1)
	if spb != 88200 {
		t.Fatalf("expected 88200 samples/bar at 90 BPM 3/4, got %d", spb)
	}

	samples := make([]float64, spb*3)
	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         90,
		LoopBars:    1,
		BeatsPerBar: 3,
		Prefix:      "t",
		RootNote:    "C3",
	})

	if len(slices) != 3 {
		t.Fatalf("expected 3 slices, got %d", len(slices))
	}
}

func TestSliceByGrid_ShortTailDiscarded(t *testing.T) {
	spb := samplesPerSliceAt(120, 4, 1)
	// Tail of 100 samples — well under half a bar (44100)
	samples := make([]float64, spb*4+100)

	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	if len(slices) != 4 {
		t.Fatalf("expected short tail to be discarded, got %d slices", len(slices))
	}
}

func TestSliceByGrid_HalfBarTailKept(t *testing.T) {
	spb := samplesPerSliceAt(120, 4, 1)
	halfBar := 44100 // samplesPerBar/2 = 88200/2
	samples := make([]float64, spb*4+halfBar)

	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	if len(slices) != 5 {
		t.Fatalf("expected half-bar tail to be kept, got %d slices", len(slices))
	}
	last := slices[4]
	if last.EndSample != len(samples) {
		t.Errorf("last slice EndSample=%d, want %d", last.EndSample, len(samples))
	}
}

func TestSliceByGrid_EmptyBuffer(t *testing.T) {
	slices := SliceByGrid(nil, testSampleRate, GridConfig{BPM: 120, RootNote: "C3"})
	if slices != nil {
		t.Fatalf("expected nil for empty buffer, got %v", slices)
	}
}

func TestSliceByGrid_ZeroBPM(t *testing.T) {
	samples := make([]float64, 44100)
	slices := SliceByGrid(samples, testSampleRate, GridConfig{BPM: 0, RootNote: "C3"})
	if slices != nil {
		t.Fatalf("expected nil for zero BPM, got %v", slices)
	}
}

func TestSliceByGrid_NoteAssignment(t *testing.T) {
	spb := samplesPerSliceAt(120, 4, 1)
	samples := make([]float64, spb*3)

	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	want := []string{"C3", "C#3", "D3"}
	for i, s := range slices {
		if s.Note != want[i] {
			t.Errorf("slice %d: note=%q, want %q", i, s.Note, want[i])
		}
	}
}

func TestSliceByGrid_TransientFields(t *testing.T) {
	spb := samplesPerSliceAt(120, 4, 1)
	samples := make([]float64, spb*2)

	slices := SliceByGrid(samples, testSampleRate, GridConfig{
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
		Prefix:      "t",
		RootNote:    "C3",
	})

	for i, s := range slices {
		wantPos := i * spb
		wantSec := float64(wantPos) / float64(testSampleRate)
		if s.Transient.SamplePos != wantPos {
			t.Errorf("slice %d: Transient.SamplePos=%d, want %d", i, s.Transient.SamplePos, wantPos)
		}
		if s.Transient.TimeSec != wantSec {
			t.Errorf("slice %d: Transient.TimeSec=%f, want %f", i, s.Transient.TimeSec, wantSec)
		}
		if s.Transient.Energy != 0 {
			t.Errorf("slice %d: Transient.Energy=%f, want 0", i, s.Transient.Energy)
		}
	}
}
