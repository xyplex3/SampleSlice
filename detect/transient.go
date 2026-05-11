// Package detect implements energy-based onset detection for mono audio.
// It locates transient events (hits, attacks) in a normalized float64 sample
// buffer and returns them sorted by time.
package detect

import (
	"cmp"
	"math"
	"slices"
)

// Transient represents a detected transient (hit/onset) in the audio.
type Transient struct {
	SamplePos int     // Sample index where the transient occurs
	TimeSec   float64 // Time in seconds
	Energy    float64 // Normalized energy at the transient point
}

// DetectTransients performs energy-based onset detection on mono audio samples.
//
// Parameters:
//   - samples: mono audio samples normalized to [-1, 1]
//   - sampleRate: audio sample rate in Hz
//   - sensitivity: detection threshold from 0.0 (very sensitive, more hits)
//     to 1.0 (very strict, fewer hits)
//   - minIntervalMs: minimum time between detected transients in milliseconds
//     (avoids double-triggering)
//   - windowSizeMs: energy analysis window in milliseconds; 0 uses the 10ms default
//
// Returns a sorted slice of Transient structs.
func DetectTransients(samples []float64, sampleRate uint32, sensitivity float64, minIntervalMs int, windowSizeMs int) []Transient {
	if len(samples) == 0 {
		return nil
	}

	// Step 1: Compute local energy using a sliding window
	wsMs := windowSizeMs
	if wsMs <= 0 {
		wsMs = 10 // default 10ms
	}
	windowSize := int(uint32(wsMs) * sampleRate / 1000)
	if windowSize < 1 {
		windowSize = 1
	}

	energy := computeEnergy(samples, windowSize)

	// Step 2: Compute onset strength by comparing each frame to the surrounding context
	onsetStrength := computeOnsetStrength(energy)

	// Step 3: Adaptive thresholding based on sensitivity
	threshold := computeAdaptiveThreshold(onsetStrength, sensitivity)

	// Step 4: Pick transients above threshold with minimum interval enforcement
	minIntervalSamples := (minIntervalMs * int(sampleRate)) / 1000
	if minIntervalSamples < 1 {
		minIntervalSamples = 1
	}

	return pickTransients(onsetStrength, energy, threshold, minIntervalSamples, windowSize, sampleRate)
}

// computeEnergy calculates the RMS energy in overlapping windows.
func computeEnergy(samples []float64, windowSize int) []float64 {
	hopSize := windowSize / 2 // 50% overlap
	if hopSize < 1 {
		hopSize = 1
	}

	numFrames := (len(samples)-windowSize)/hopSize + 1
	if numFrames <= 0 {
		numFrames = 1
	}

	energy := make([]float64, numFrames)
	for i := range numFrames {
		start := i * hopSize
		end := start + windowSize
		if end > len(samples) {
			end = len(samples)
		}

		sum := 0.0
		for j := start; j < end; j++ {
			sum += samples[j] * samples[j]
		}

		energy[i] = math.Sqrt(sum / float64(end-start))
	}

	return energy
}

// computeOnsetStrength calculates how much each frame's energy exceeds the local context.
func computeOnsetStrength(energy []float64) []float64 {
	if len(energy) == 0 {
		return nil
	}

	const contextSize = 10 // look back 10 frames
	onsetStrength := make([]float64, len(energy))

	for i := range len(energy) {
		start := max(0, i-contextSize)

		sum := 0.0
		for j := start; j < i; j++ {
			sum += energy[j]
		}

		avgEnergy := 0.0
		if i > start {
			avgEnergy = sum / float64(i-start)
		}

		onsetStrength[i] = energy[i] - avgEnergy
		if onsetStrength[i] < 0 {
			onsetStrength[i] = 0
		}
	}

	// Normalize onset strength to [0, 1]
	maxStrength := slices.Max(onsetStrength)
	if maxStrength > 0 {
		for i := range onsetStrength {
			onsetStrength[i] /= maxStrength
		}
	}

	return onsetStrength
}

// computeAdaptiveThreshold determines the detection threshold based on sensitivity.
// sensitivity 0.0 -> threshold 0.05 (very sensitive)
// sensitivity 1.0 -> threshold 0.85 (very strict)
func computeAdaptiveThreshold(onsetStrength []float64, sensitivity float64) float64 {
	threshold := 0.05 + sensitivity*0.80

	// For very quiet signals (max onset strength below 0.1), lower the floor so
	// at least some transients are detected. Don't cap high-sensitivity thresholds
	// downward — that would make sensitivity ineffective on drum-heavy material.
	if len(onsetStrength) > 0 {
		maxStrength := slices.Max(onsetStrength)
		if maxStrength > 0 && threshold > maxStrength {
			threshold = maxStrength * 0.9
		}
	}

	return threshold
}

// pickTransients selects transient positions above the threshold with minimum interval enforcement.
func pickTransients(onsetStrength []float64, energy []float64, threshold float64, minIntervalSamples int, windowSize int, sampleRate uint32) []Transient {
	type candidate struct {
		frameIndex int
		strength   float64
	}

	var candidates []candidate
	for i, s := range onsetStrength {
		if s > threshold {
			candidates = append(candidates, candidate{frameIndex: i, strength: s})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	slices.SortFunc(candidates, func(a, b candidate) int {
		return cmp.Compare(b.strength, a.strength) // descending by strength
	})

	hopSize := windowSize / 2
	if hopSize < 1 {
		hopSize = 1
	}

	var transients []Transient
	for _, c := range candidates {
		samplePos := c.frameIndex * hopSize

		tooClose := false
		for _, t := range transients {
			diff := samplePos - t.SamplePos
			if diff < 0 {
				diff = -diff
			}
			if diff < minIntervalSamples {
				tooClose = true
				break
			}
		}

		if !tooClose {
			peakFrame := findLocalPeak(energy, c.frameIndex, 5)
			peakSample := peakFrame * hopSize

			transients = append(transients, Transient{
				SamplePos: peakSample,
				TimeSec:   float64(peakSample) / float64(sampleRate),
				Energy:    energy[peakFrame],
			})
		}
	}

	slices.SortFunc(transients, func(a, b Transient) int {
		return cmp.Compare(a.SamplePos, b.SamplePos)
	})

	return transients
}

// findLocalPeak finds the frame with maximum energy near a given position.
func findLocalPeak(energy []float64, center, searchRange int) int {
	start := max(0, center-searchRange)
	end := center + searchRange
	if end >= len(energy) {
		end = len(energy) - 1
	}

	peakIdx := start
	for i := start + 1; i <= end; i++ {
		if energy[i] > energy[peakIdx] {
			peakIdx = i
		}
	}

	return peakIdx
}

