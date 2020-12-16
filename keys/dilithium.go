package keys

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/theQRL/go-qrllib-crypto/dilithium"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
)

type DilithiumKeys struct {
	outFileName string

	pbData *protos.DilithiumKeys
}

func (dk *DilithiumKeys) GetDilithiumGroup() []*protos.DilithiumGroup {
	return dk.pbData.DilithiumGroup
}

func (dk *DilithiumKeys) Add(dilithiumGroup *protos.DilithiumGroup) {
	d := dilithium.NewDilithium()
	dilithiumInfo := &protos.DilithiumInfo{
		PK: hex.EncodeToString(d.PK()),
		SK: hex.EncodeToString(d.SK()),
	}
	dilithiumGroup.DilithiumInfo = append(dilithiumGroup.DilithiumInfo, dilithiumInfo)

	fmt.Println("Added New Dilithium Address")
}

func (dk *DilithiumKeys) AddGroup(dilithiumGroup *protos.DilithiumGroup) {
	dk.pbData.DilithiumGroup = append(dk.pbData.DilithiumGroup, dilithiumGroup)
	dk.Save()
}

func (dk *DilithiumKeys) List() {
	for i, dilithiumGroup := range dk.pbData.DilithiumGroup {
		for _, dilithiumInfo := range dilithiumGroup.DilithiumInfo{
			fmt.Println(fmt.Sprintf("Group #%d\tPK: %s\tSK: %s", i, dilithiumInfo.PK, dilithiumInfo.SK))
		}
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

func (dk *DilithiumKeys) GetDilithiumGroupByIndex(index uint) (*protos.DilithiumGroup, error) {
	if int(index) > len(dk.pbData.DilithiumGroup) {
		return nil, errors.New(fmt.Sprintf("Invalid Dilithium Group Index"))
	}
	return dk.pbData.DilithiumGroup[index - 1], nil
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
