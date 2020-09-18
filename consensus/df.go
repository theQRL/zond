package consensus

import (
	"crypto/sha256"
)

func ComputeDelayedFunction(input []byte, numberOfElements int) [][]byte {
	computationPerNode := 10000

	var result [][]byte
	result = append(result, input)
	for element := 0; element < numberOfElements; element++ {
		for i := 0; i < computationPerNode; i++ {
			h := sha256.New()
			h.Write(input)
			input = h.Sum(nil)
		}
		result = append(result, input)
	}

	return result
}
