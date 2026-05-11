package slice_test

import (
	"slices"
	"testing"

	"sampleslice/slice"
)

func TestTrimSilence_LeadingTrimmed(t *testing.T) {
	// Leading zeros removed; trailing non-silent sample kept.
	in := []float64{0.0, 0.0, 0.5, 0.3}
	got := slice.TrimSilence(in, 0.001)
	want := []float64{0.5, 0.3}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrimSilence_TrailingTrimmed(t *testing.T) {
	in := []float64{0.5, 0.3, 0.0, 0.0}
	got := slice.TrimSilence(in, 0.001)
	want := []float64{0.5, 0.3}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrimSilence_BothEndsTrimmed(t *testing.T) {
	in := []float64{0.0, 0.0, 0.7, 0.8, 0.0, 0.0}
	got := slice.TrimSilence(in, 0.001)
	want := []float64{0.7, 0.8}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrimSilence_EntirelySilent(t *testing.T) {
	in := []float64{0.0, 0.0, 0.0}
	got := slice.TrimSilence(in, 0.001)
	if !slices.Equal(got, in) {
		t.Errorf("got %v, want input unchanged %v", got, in)
	}
}

func TestTrimSilence_NoSilence(t *testing.T) {
	in := []float64{0.5, 0.6, 0.7}
	got := slice.TrimSilence(in, 0.001)
	if !slices.Equal(got, in) {
		t.Errorf("got %v, want input unchanged %v", got, in)
	}
}

func TestTrimSilence_CustomNoiseFloor(t *testing.T) {
	// Sample exactly at noiseFloor is treated as silence
	in := []float64{0.0, 0.05, 0.5, 0.05, 0.0}
	got := slice.TrimSilence(in, 0.05)
	want := []float64{0.5}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
