package keys

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"reflect"
)

type DilithiumKeys struct {
	outFileName string

	pbData *protos.DilithiumKeys
}

func (dk *DilithiumKeys) GetDilithiumInfo() []*protos.DilithiumInfo {
	return dk.pbData.DilithiumInfo
}

func (dk *DilithiumKeys) Add(dilithiumAccount *dilithium.Dilithium) {
	pk := dilithiumAccount.GetPK()
	strPK := hex.EncodeToString(pk[:])
	sk := dilithiumAccount.GetSK()
	strSK := hex.EncodeToString(sk[:])
	hexSeed := dilithiumAccount.GetHexSeed()

	found := false
	for _, info := range dk.pbData.DilithiumInfo {
		if reflect.DeepEqual(info.PK, strPK) {
			found = true
		}
	}

	if found {
		fmt.Println("Dilithium Key already exists")
		return
	}

	dk.pbData.DilithiumInfo = append(dk.pbData.DilithiumInfo, &protos.DilithiumInfo{
		PK:      strPK,
		SK:      strSK,
		HexSeed: hexSeed,
	})

	fmt.Println("Added Dilithium Key")
	dk.Save()
}

func (dk *DilithiumKeys) List() {
	for i, dilithiumInfo := range dk.pbData.DilithiumInfo {
		fmt.Println(fmt.Sprintf("Index #%d\tPK: %s\tSK: %s", i, dilithiumInfo.PK, dilithiumInfo.SK))
	}
}

func (dk *DilithiumKeys) Remove() {

	dk.Save()
}

func (dk *DilithiumKeys) Save() {
	jsonData, err := protojson.Marshal(dk.pbData)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	err = ioutil.WriteFile(dk.outFileName, jsonData, 0644)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	return
}

func (dk *DilithiumKeys) GetDilithiumByIndex(index uint) (*protos.DilithiumInfo, error) {
	if int(index) > len(dk.pbData.DilithiumInfo) {
		return nil, errors.New(fmt.Sprintf("Invalid Dilithium Group Index"))
	}
	return dk.pbData.DilithiumInfo[index-1], nil
}

func (dk *DilithiumKeys) Load() {
	dk.pbData = &protos.DilithiumKeys{}

	if !misc.FileExists(dk.outFileName) {
		return
	}
	data, err := ioutil.ReadFile(dk.outFileName)
	if err != nil {
		fmt.Println("Error reading file: ", err)
		return
	}
	err = protojson.Unmarshal(data, dk.pbData)
	if err != nil {
		fmt.Println("Error while decoding file: ", err)
		return
	}
}

func NewDilithiumKeys(outFileName string) *DilithiumKeys {
	w := &DilithiumKeys{
		outFileName: outFileName,
	}
	w.Load()

	return w
}
