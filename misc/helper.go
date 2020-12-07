package misc

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"math"
	"os"
	"runtime"
	"strconv"
)

type UcharVector struct {
	data goqrllib.UcharVector
}

func NewUCharVector() *UcharVector {
	u := &UcharVector{}
	u.data = goqrllib.NewUcharVector()

	// Finalizer to clean up memory allocated by C++ when object becomes unreachable
	runtime.SetFinalizer(u,
		func(u *UcharVector) {
			goqrllib.DeleteUcharVector(u.data)
		})
	return u
}

func (v *UcharVector) AddBytes(data []byte) {
	for _, element := range data {
		v.data.Add(element)
	}
}

func (v *UcharVector) AddByte(data byte) {
	v.data.Add(data)
}

func (v *UcharVector) GetBytesBuffer() bytes.Buffer {
	var data bytes.Buffer
	for i := int64(0); i < v.data.Size(); i++ {
		value := v.data.Get(int(i))
		data.WriteByte(value)
	}
	return data
}

func (v *UcharVector) GetBytes() []byte {
	data := v.GetBytesBuffer()
	return data.Bytes()
}

func (v *UcharVector) GetString() string {
	data := v.GetBytesBuffer()
	return data.String()
}

func (v *UcharVector) GetData() goqrllib.UcharVector {
	return v.data
}

func (v *UcharVector) New(data goqrllib.UcharVector) {
	if v.data != nil {
		goqrllib.DeleteUcharVector(v.data)
	}
	v.data = data
}

func BytesToUCharVector(data []byte) *UcharVector {
	u := NewUCharVector()
	for _, element := range data {
		u.AddByte(element)
	}

	return u
}

func Int64ToUCharVector(data int64) *UcharVector {
	u := NewUCharVector()
	u.New(goqrllib.NewUcharVector__SWIG_1(data))
	return u
}

func UCharVectorToBytes(data goqrllib.UcharVector) []byte {
	vector := NewUCharVector()
	vector.New(data)

	return vector.GetBytes()
}

func UCharVectorToString(data goqrllib.UcharVector) string {
	return string(UCharVectorToBytes(data))
}

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
				tmp := NewUCharVector()
				tmp.AddBytes(e.Value.([]byte))
				e = e.Next()
				tmp.AddBytes(e.Value.([]byte))
				e = e.Next()
				h := sha256.New()
				h.Write(tmp.GetBytes())
				nextLayer.PushBack(h.Sum(nil))
			}
			z += 2
		}
		lArray = nextLayer
	}
	return lArray.Back().Value.([]byte)
}

func Reverse(s [][]byte) [][]byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	return s
}

func Address2Bin(qaddress string) ([]byte, error) {
	return hex.DecodeString(qaddress)
}

func Bin2Address(binAddress []byte) string {
	return hex.EncodeToString(binAddress)
}

func Bin2Addresses(binAddresses [][]byte) []string {
	addresses := make([]string, 0)
	for i := 0 ; i < len(binAddresses); i++ {
		addresses = append(addresses, Bin2Address(binAddresses[i]))
	}
	return addresses
}

func Bin2Pks(binPks [][]byte) []string {
	pks := make([]string, 0)
	for i := 0 ; i < len(binPks); i++ {
		pks = append(pks, hex.EncodeToString(binPks[i]))
	}
	return pks
}

func PK2BinAddress(pk []byte) []byte {
	return UCharVectorToBytes(goqrllib.QRLHelperGetAddress(BytesToUCharVector(pk).GetData()))
}

func PK2Qaddress(pk []byte) string {
	return Bin2Address(PK2BinAddress(pk))
}

func ConvertBytesToLong(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func OTSKeyFromSig(signature []byte) uint64 {
	return uint64(binary.BigEndian.Uint32(signature[0:4]))
}

func BytesToString(data []byte) string {
	buff := bytes.NewBuffer(data)
	return buff.String()
}

func Sha256(message string, length int64) []byte {
	return UCharVectorToBytes(goqrllib.Sha2_256_n(BytesToUCharVector([]byte(message)).GetData(), length))
}

func StringAddressToBytesArray(addrs []string) ([][]byte, error) {
	bytesAddrs := make([][]byte, len(addrs))
	var err error
	for i := 0; i < len(addrs); i++ {
		bytesAddrs[i], err = Address2Bin(addrs[i])
		if err != nil {
			return nil, err
		}
	}

	return bytesAddrs, nil
}

func UInt64ToString(data []uint64) []string {
	convertedData := make([]string, len(data))
	for i := range data {
		convertedData[i] = strconv.FormatUint(data[i], 10)
	}
	return convertedData
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

func IsValidAddress(address []byte) bool {
	// Warning: Never pass this validation True for Coinbase Address
	if goqrllib.QRLHelperAddressIsValid(BytesToUCharVector(address).GetData()) {
		return true
	}
	return false
}