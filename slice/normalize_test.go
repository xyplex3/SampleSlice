package slice_test

import (
	"math"
	"testing"

	"sampleslice/slice"
)

func TestNormalizeSlice_QuietAmplified(t *testing.T) {
	in := []float64{0.0, 0.25, -0.25, 0.1}
	got := slice.NormalizeSlice(in)
	// peak = 0.25, gain = 4.0 → max should be 1.0
	wantPeak := 1.0
	peak := 0.0
	for _, s := range got {
		if math.Abs(s) > peak {
			peak = math.Abs(s)
		}
	}
	if math.Abs(peak-wantPeak) > 1e-9 {
		t.Errorf("peak = %v, want %v", peak, wantPeak)
	}
}

func TestNormalizeSlice_FullScaleUnchanged(t *testing.T) {
	in := []float64{0.5, 1.0, -0.5}
	got := slice.NormalizeSlice(in)
	for i, s := range got {
		if s != in[i] {
			t.Errorf("index %d: got %v, want %v (unchanged)", i, s, in[i])
		}
	}
}

func TestNormalizeSlice_SilenceUnchanged(t *testing.T) {
	in := []float64{0.0, 0.0, 0.0}
	got := slice.NormalizeSlice(in)
	for i, s := range got {
		if s != in[i] {
			t.Errorf("index %d: got %v, want %v", i, s, in[i])
		}
	}
}

func TestNormalizeSlice_NoSampleOutOfRange(t *testing.T) {
	in := []float64{0.1, -0.2, 0.05, -0.3}
	got := slice.NormalizeSlice(in)
	for i, s := range got {
		if math.Abs(s) > 1.0+1e-9 {
			t.Errorf("index %d: sample %v exceeds [-1,1]", i, s)
		}
	}
}
