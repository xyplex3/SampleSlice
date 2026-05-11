package slice

import "math"

// DeduplicateSlices removes slices whose audio content is very similar to a
// previously accepted slice. Similarity is measured by cosine distance on an
// RMS energy envelope (16 equal-width windows across the slice).
//
// threshold controls strictness: 0 = disabled (keep all), values approaching 1
// remove near-duplicates more aggressively. 0.95 is a reasonable starting point
// for drum machine material.
func DeduplicateSlices(slices []AudioSlice, samples []float64, threshold float64) []AudioSlice {
	if threshold <= 0 || len(slices) == 0 {
		return slices
	}

	const numBins = 16
	result := make([]AudioSlice, 0, len(slices))
	accepted := make([][]float64, 0, len(slices))

	for _, s := range slices {
		profile := computeEnergyProfile(ExtractSamples(samples, s), numBins)

		duplicate := false
		for _, existing := range accepted {
			if cosineSimilarity(profile, existing) >= threshold {
				duplicate = true
				break
			}
		}

		if !duplicate {
			result = append(result, s)
			accepted = append(accepted, profile)
		}
	}

	return result
}

// computeEnergyProfile divides samples into numBins equal windows and returns
// the RMS energy of each window as a feature vector.
func computeEnergyProfile(samples []float64, numBins int) []float64 {
	profile := make([]float64, numBins)
	n := len(samples)
	if n == 0 {
		return profile
	}

	for i := 0; i < numBins; i++ {
		start := (i * n) / numBins
		end := ((i + 1) * n) / numBins
		if end > n {
			end = n
		}
		if start >= end {
			continue
		}

		var sumSq float64
		for _, v := range samples[start:end] {
			sumSq += v * v
		}
		profile[i] = math.Sqrt(sumSq / float64(end-start))
	}

	return profile
}

// cosineSimilarity returns the cosine similarity in [0,1] between two vectors.
// Returns 0 for zero vectors or mismatched lengths.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
