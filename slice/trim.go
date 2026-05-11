package slice

// TrimSilence removes leading and trailing samples below noiseFloor.
// noiseFloor is a linear amplitude threshold (e.g. 0.001 ≈ -60 dBFS).
// Returns the input slice unchanged if it is entirely silent.
func TrimSilence(samples []float64, noiseFloor float64) []float64 {
	first := -1
	for i, s := range samples {
		if s > noiseFloor || s < -noiseFloor {
			first = i
			break
		}
	}
	if first < 0 {
		return samples
	}

	last := first
	for i := len(samples) - 1; i > first; i-- {
		if samples[i] > noiseFloor || samples[i] < -noiseFloor {
			last = i
			break
		}
	}
	return samples[first : last+1]
}
