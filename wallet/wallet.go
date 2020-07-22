package wallet

import (
	"fmt"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"os"
)

type Wallet struct {
	outFileName string

	pbData *protos.Wallet
}

func (w *Wallet) Add(height uint64, hashFunction goqrllib.EHashFunction) {
	xmss := crypto.FromHeight(height, hashFunction)
	xmssInfo := &protos.XMSSInfo{
		Address: xmss.QAddress(),
		HexSeed: xmss.HexSeed(),
		Mnemonic: xmss.Mnemonic(),
	}
	w.pbData.XmssInfo = append(w.pbData.XmssInfo, xmssInfo)

	fmt.Println("Added New Zond Address: ", xmssInfo.Address)

	w.Save()
}

func (w *Wallet) List() {
	for i, xmssInfo := range w.pbData.XmssInfo {
		fmt.Println(fmt.Sprintf("%d\t%s", i, xmssInfo.Address))
	}
}

func (w *Wallet) Remove() {

	w.Save()
}

func (w *Wallet) Save() {
	jsonData, err := protojson.Marshal(w.pbData)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	err = ioutil.WriteFile(w.outFileName, jsonData, 0644)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	return
}

func (w *Wallet) Load() {
	w.pbData = &protos.Wallet{}

	if !fileExists(w.outFileName) {
		return
	}
	data, err := ioutil.ReadFile(w.outFileName)
	if err != nil {
		fmt.Println("Error reading file: ", err)
		return
	}
	err = protojson.Unmarshal(data, w.pbData)
	if err != nil {
		fmt.Println("Error while decoding file: ", err)
		return
	}
}

func NewWallet(outFileName string) *Wallet {
	w := &Wallet{
		outFileName: outFileName,
	}
	w.Load()

	return w
}

func fileExists(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}