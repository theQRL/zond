package misc

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	common2 "github.com/theQRL/go-qrllib/common"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/common"
	"math"
	"os"
	"strconv"
	"strings"
)

func IPFromMultiAddr(multiAddr string) string {
	info := strings.Split(multiAddr, "/")
	return info[2]
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
	convertedData := float64(data) / 1000000000 // TODO: Replace with config.dev.ShorPerQuanta
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

func Bin2Addresses(binAddresses [][]byte) []string {
	addresses := make([]string, 0)
	for i := 0; i < len(binAddresses); i++ {
		addresses = append(addresses, BytesToHexStr(binAddresses[i]))
	}
	return addresses
}

func StringAddressToBytesArray(addrs []string) ([][]byte, error) {
	bytesAddrs := make([][]byte, len(addrs))
	var err error
	for i := 0; i < len(addrs); i++ {
		bytesAddrs[i], err = HexStrToBytes(addrs[i])
		if err != nil {
			return nil, err
		}
	}

	return bytesAddrs, nil
}

func UnSizedXMSSPKToSizedPK(pk []byte) (pkSized [xmss.ExtendedPKSize]uint8) {
	copy(pkSized[:], pk)
	return
}

func UnSizedDilithiumPKToSizedPK(pk []byte) (pkSized [dilithium.PKSizePacked]uint8) {
	copy(pkSized[:], pk)
	return
}

func UnSizedAddressToSizedAddress(addr []byte) (addrSized [common2.AddressSize]uint8) {
	copy(addrSized[:], addr)
	return
}

func GetMaxOTSBitfieldSizeFromAddress(addr common.Address) uint64 {
	height := (addr[1] & 0xf) << 1
	maxOTS := 2 << height
	maxOTSBitfieldSize := uint64(math.Ceil(float64(maxOTS) / 8.0))
	return maxOTSBitfieldSize
}

func GetOTSIndexFromSignature(signature []byte) uint64 {
	return uint64(binary.BigEndian.Uint32(signature[0:4]))
}

func IsUsedOTSIndex(otsIndex uint64, otsBitfield [][8]byte) bool {
	offset := otsIndex >> 3
	// otsBitfield can only be nil if it has not made any transaction yet
	if otsBitfield == nil || len(otsBitfield) == 0 {
		return false
	}
	if offset >= uint64(len(otsBitfield)) {
		return true
	}

	relative := otsIndex % 8

	return otsBitfield[offset][relative] == 1
}

func GetSignatureType(addr common.Address) common2.SignatureType {
	return common2.SignatureType(addr[0])
}

func GetWalletTypeFromPK(pk []byte) common2.SignatureType {
	if len(pk) == dilithium.PKSizePacked {
		return common2.DilithiumSig
	} else if len(pk) == xmss.ExtendedPKSize {
		return common2.XMSSSig
	} else {
		panic("invalid signature type")
	}
}

func GetDilithiumAddressFromUnSizedPK(pk []byte) common.Address {
	addrOutput := dilithium.GetDilithiumAddressFromPK(UnSizedDilithiumPKToSizedPK(pk))
	var address common.Address
	copy(address[:], addrOutput[:])

	return address
}

func GetXMSSAddressFromUnSizedPK(pk []byte) common.Address {
	addrOutput := xmss.GetXMSSAddressFromPK(UnSizedXMSSPKToSizedPK(pk))
	var address common.Address
	copy(address[:], addrOutput[:])

	return address
}

func GetAddressFromUnSizedPK(pk []byte) common.Address {
	if len(pk) == xmss.ExtendedPKSize {
		return GetXMSSAddressFromUnSizedPK(pk)
	} else if len(pk) == dilithium.PKSizePacked {
		return GetDilithiumAddressFromUnSizedPK(pk)
	}
	return common.Address{}
}

func ClearPrefix0x(data string) string {
	if strings.HasPrefix(data, "0x") {
		return data[2:]
	}
	return data
}

func BytesToHexStr(data []uint8) string {
	return "0x" + hex.EncodeToString(data)
}

func HexStrToBytes(data string) ([]uint8, error) {
	return hex.DecodeString(ClearPrefix0x(data))
}
