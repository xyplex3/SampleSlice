package slice

// NormalizeSlice peak-normalizes samples to 0 dBFS.
// Silent slices (peak == 0) and already clipped slices (peak >= 1.0) are returned unchanged.
func NormalizeSlice(samples []float64) []float64 {
	peak := 0.0
	for _, s := range samples {
		if s < 0 {
			s = -s
		}
		if s > peak {
			peak = s
		}
	}
	if peak == 0 || peak >= 1.0 {
		return samples
	}
	gain := 1.0 / peak
	out := make([]float64, len(samples))
	for i, s := range samples {
		out[i] = s * gain
	}
	return out
}
