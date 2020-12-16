package misc

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"math"
	"os"
	"strconv"
)

func MerkleTXHash(hashes list.List) []byte {
	j := int(math.Ceil(math.Log2(float64(hashes.Len()))))

	lArray := hashes
	for x := 0; x < j; x++ {
		var nextLayer list.List
		h := lArray
		i := h.Len()%2 + h.Len()/2
		e := h.Front()
		z := 0

		for k := 0; k < i; k++ {
			if h.Len() == z+1 {
				nextLayer.PushBack(e.Value.([]byte))
			} else {
				h := sha256.New()
				h.Write(e.Value.([]byte))
				e = e.Next()
				h.Write(e.Value.([]byte))
				e = e.Next()
				nextLayer.PushBack(h.Sum(nil))
			}
			z += 2
		}
		lArray = nextLayer
	}
	return lArray.Back().Value.([]byte)
}

func ConvertBytesToLong(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func BytesToString(data []byte) string {
	buff := bytes.NewBuffer(data)
	return buff.String()
}

func ShorToQuanta(data uint64) string {
	convertedData := float64(data) / 1000000000  // TODO: Replace with config.dev.ShorPerQuanta
	return strconv.FormatFloat(convertedData, 'f', 9, 64)
}

func ShorsToQuantas(data []uint64) []string {
	convertedData := make([]string, len(data))
	for i := range data {
		convertedData[i] = ShorToQuanta(data[i])
	}
	return convertedData
}

func FileExists(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
