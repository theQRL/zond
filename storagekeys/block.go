package storagekeys

import "strconv"

func GetBlockHashStorageKeyBySlotNumber(slotNumber uint64) []byte {
	return []byte(strconv.FormatUint(slotNumber, 10))
}
