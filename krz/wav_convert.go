package krz

// ConvertTo8BitUnsigned converts 16-bit signed PCM samples to 8-bit unsigned format.
// 16-bit signed range: -32768 to 32767
// 8-bit unsigned range: 0 to 255 (128 = silence)
func ConvertTo8BitUnsigned(input []int16) []uint8 {
	result := make([]uint8, len(input))
	for i, sample := range input {
		result[i] = uint8((int32(sample) + 32768) >> 8)
	}
	return result
}

// ConvertWAVToKRZFormat converts raw WAV sample data to the KRZ sample format.
// bitsPerSample determines the output format:
//   - 8: 8-bit unsigned PCM (format 0)
//   - 16: 16-bit signed big-endian PCM (format 2)
//   - 0: default to 8-bit unsigned
func ConvertWAVToKRZFormat(input []int16, bitsPerSample int) []byte {
	switch bitsPerSample {
	case 16:
		// Convert to 16-bit signed big-endian
		result := make([]byte, len(input)*2)
		for i, sample := range input {
			result[i*2] = byte((int32(sample) >> 8) & 0xFF)
			result[i*2+1] = byte(int32(sample) & 0xFF)
		}
		return result
	default:
		// Default to 8-bit unsigned
		converted := ConvertTo8BitUnsigned(input)
		return []byte(converted)
	}
}

// CompressADPCM applies Kurzweil ADPCM compression to 8-bit unsigned sample data.
// The compression encodes 2 samples per byte using 4-bit delta values.
func CompressADPCM(input []uint8) []uint8 {
	if len(input) == 0 {
		return nil
	}

	// Output: 2 samples per byte, so roughly half the input size
	outputLen := (len(input) + 1) / 2
	result := make([]uint8, outputLen)

	// Step size index and current predicted value
	stepIndex := 0
	predicted := int32(input[0])

	// ADPCM step size table (simplified version matching Kurzweil behavior)
	stepTable := [16]uint8{
		7, 8, 9, 10, 11, 12, 13, 14,
		16, 18, 21, 24, 28, 32, 37, 43,
	}

	// Index adjustment table based on delta magnitude
	indexAdjust := [16]int{
		0, 0, 0, 0, 1, 1, 1, 1,
		2, 2, 2, 2, 3, 3, 3, 3,
	}

	for i := 0; i < len(input); i += 2 {
		byteIdx := i / 2
		highDelta := encodeDelta(input[i], predicted, stepTable[stepIndex])
		predicted = updatePrediction(predicted, highDelta, stepTable[stepIndex])
		stepIndex = updateStepIndex(stepIndex, highDelta, indexAdjust)

		lowDelta := int32(0)
		if i+1 < len(input) {
			lowDelta = encodeDelta(input[i+1], predicted, stepTable[stepIndex])
			predicted = updatePrediction(predicted, lowDelta, stepTable[stepIndex])
			stepIndex = updateStepIndex(stepIndex, lowDelta, indexAdjust)
		}

		result[byteIdx] = (uint8(highDelta&0xF) << 4) | (uint8(lowDelta & 0xF))
	}

	return result
}

// encodeDelta computes a 4-bit delta value between actual and predicted sample
func encodeDelta(actual uint8, predicted int32, step uint8) int32 {
	actualInt := int32(actual)
	diff := actualInt - predicted

	// Clamp and scale to 4-bit range (0-15)
	if diff < 0 {
		diff = -diff
	}

	delta := (diff * 15) / int32(step)
	if delta > 15 {
		delta = 15
	}

	// Preserve sign in the 4-bit encoding
	if int32(actual) < predicted {
		delta = -delta
	}

	return delta
}

// updatePrediction adjusts the predicted value based on delta and step
func updatePrediction(predicted int32, delta int32, step uint8) int32 {
	// Decode the 4-bit delta back to a signed value
	signedDelta := delta
	if signedDelta > 7 {
		signedDelta -= 16
	}

	newPredicted := predicted + (signedDelta * int32(step))

	// Clamp to valid 8-bit unsigned range
	if newPredicted < 0 {
		newPredicted = 0
	}
	if newPredicted > 255 {
		newPredicted = 255
	}

	return newPredicted
}

// updateStepIndex adjusts the step size index based on delta magnitude
func updateStepIndex(currentIndex int, delta int32, adjustTable [16]int) int {
	// Get absolute value of delta for index lookup
	deltaIdx := int(delta)
	if deltaIdx < 0 {
		deltaIdx = -deltaIdx
	}
	if deltaIdx > 15 {
		deltaIdx = 15
	}

	newIndex := currentIndex + adjustTable[deltaIdx]
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex > 15 {
		newIndex = 15
	}

	return newIndex
}