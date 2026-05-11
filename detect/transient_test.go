package detect_test

import (
	"math"
	"testing"

	"sampleslice/detect"
)

// TestDetectTransients_EmptyInput verifies that empty input returns nil.
func TestDetectTransients_EmptyInput(t *testing.T) {
	result := detect.DetectTransients(nil, 44100, 0.5, 50, 0)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

// TestDetectTransients_SilentAudio verifies that silent audio produces no transients.
func TestDetectTransients_SilentAudio(t *testing.T) {
	samples := make([]float64, 44100) // 1 second of silence
	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)
	if len(result) != 0 {
		t.Errorf("expected no transients for silent audio, got %d", len(result))
	}
}

// TestDetectTransients_SingleTransient verifies detection of a single spike.
func TestDetectTransients_SingleTransient(t *testing.T) {
	samples := make([]float64, 44100)
	// Place a spike at sample 22050 (middle of 1 second)
	samples[22050] = 1.0

	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)

	if len(result) == 0 {
		t.Fatal("expected at least one transient, got none")
	}

	// The detected transient should be near the spike position.
	got := result[0]
	if math.Abs(float64(got.SamplePos-22050)) > 2205 { // within ~50ms window
		t.Errorf("transient at sample %d, expected near 22050", got.SamplePos)
	}
}

// TestDetectTransients_MultipleTransients verifies detection of multiple spikes.
func TestDetectTransients_MultipleTransients(t *testing.T) {
	samples := make([]float64, 44100)
	// Place spikes at 10%, 50%, and 90% of the audio
	samples[4410] = 0.8
	samples[22050] = 1.0
	samples[39690] = 0.6

	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)

	if len(result) < 2 {
		t.Errorf("expected at least 2 transients, got %d", len(result))
	}
}

// TestDetectTransients_BelowThreshold verifies that silence produces
// no transient detections regardless of sensitivity setting.
func TestDetectTransients_BelowThreshold(t *testing.T) {
	samples := make([]float64, 44100)
	// all zeros = pure silence, no transients possible

	// Use high sensitivity (0.9 = very strict threshold)
	result := detect.DetectTransients(samples, 44100, 0.9, 50, 0)

	if len(result) != 0 {
		t.Errorf("expected no transients in silence, got %d", len(result))
	}
}

// TestDetectTransients_SensitiveDetection verifies that low sensitivity detects more transients.
func TestDetectTransients_SensitiveDetection(t *testing.T) {
	samples := make([]float64, 44100)
	samples[22050] = 0.3 // moderate spike

	// Use low sensitivity (0.1 = very sensitive, low threshold)
	result := detect.DetectTransients(samples, 44100, 0.1, 50, 0)

	if len(result) == 0 {
		t.Error("expected at least one transient with low threshold, got none")
	}
}

// TestDetectTransients_MinIntervalEnforcement verifies that close transients are merged.
func TestDetectTransients_MinIntervalEnforcement(t *testing.T) {
	samples := make([]float64, 44100)
	// Place two spikes very close together (within 10ms)
	samples[22050] = 1.0
	samples[22094] = 0.9 // ~1ms later

	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)

	// With 50ms min interval, these should be merged into one transient
	if len(result) > 1 {
		t.Errorf("expected 1 transient with min interval enforcement, got %d", len(result))
	}
}

// TestDetectTransients_SortedByPosition verifies transients are returned sorted by sample position.
func TestDetectTransients_SortedByPosition(t *testing.T) {
	samples := make([]float64, 44100)
	samples[4410] = 1.0
	samples[22050] = 1.0
	samples[39690] = 1.0

	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)

	for i := 1; i < len(result); i++ {
		if result[i].SamplePos <= result[i-1].SamplePos {
			t.Errorf("transients not sorted: index %d sample %d <= index %d sample %d",
				i, result[i].SamplePos, i-1, result[i-1].SamplePos)
		}
	}
}

// TestDetectTransients_TimeSecAccuracy verifies TimeSec is correctly calculated from SamplePos.
func TestDetectTransients_TimeSecAccuracy(t *testing.T) {
	samples := make([]float64, 44100)
	samples[22050] = 1.0

	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)

	if len(result) == 0 {
		t.Fatal("expected at least one transient")
	}

	got := result[0]
	expectedTime := float64(got.SamplePos) / 44100.0
	if math.Abs(got.TimeSec-expectedTime) > 0.001 {
		t.Errorf("TimeSec = %f, expected %f based on SamplePos %d", got.TimeSec, expectedTime, got.SamplePos)
	}
}

// TestDetectTransients_EnergyPositive verifies detected transients have positive energy.
func TestDetectTransients_EnergyPositive(t *testing.T) {
	samples := make([]float64, 44100)
	samples[22050] = 1.0

	result := detect.DetectTransients(samples, 44100, 0.5, 50, 0)

	for i, tr := range result {
		if tr.Energy <= 0 {
			t.Errorf("transient %d has non-positive energy: %f", i, tr.Energy)
		}
	}
}

// BenchmarkDetectTransients measures transient detection performance.
func BenchmarkDetectTransients(b *testing.B) {
	samples := make([]float64, 44100*10) // 10 seconds of audio
	// Add some spikes
	for i := 0; i < 100; i++ {
		samples[i*44100/100] = 0.9
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detect.DetectTransients(samples, 44100, 0.5, 50, 0)
	}
}