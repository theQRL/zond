package misc

import (
	"fmt"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
)

type TransactionList struct {
	outFileName string

	pbData *protos.TransactionList
}

func (t *TransactionList) GetTransactions() []*protos.Transaction {
	var tl []*protos.Transaction
	for _, tx := range t.pbData.Txs {
		tl = append(tl, tx)
	}
	return tl
}

func (t *TransactionList) Add(tx *protos.Transaction) {
	t.pbData.Txs = append(t.pbData.Txs, tx)

	// TODO: Fix call to TransactionHash
	//fmt.Println("Added New Transaction: ", misc.BytesToHexStr(tx.TransactionHash))

	t.Save()
}

func (t *TransactionList) List() {
	// TODO: Fix call to TransactionHash
	//for i, tx := range t.pbData.Txs {
	//	fmt.Println(fmt.Sprintf("%d\t%s", i, misc.BytesToHexStr(tx.TransactionHash)))
	//}
}

func (t *TransactionList) Remove() {

	t.Save()
}

func (t *TransactionList) Save() {
	jsonData, err := protojson.Marshal(t.pbData)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	err = ioutil.WriteFile(t.outFileName, jsonData, 0644)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	return
}

func (t *TransactionList) Load() {
	t.pbData = &protos.TransactionList{}

	if !FileExists(t.outFileName) {
		return
	}
	data, err := ioutil.ReadFile(t.outFileName)
	if err != nil {
		fmt.Println("Error reading file: ", err)
		return
	}
	err = protojson.Unmarshal(data, t.pbData)
	if err != nil {
		fmt.Println("Error while decoding file: ", err)
		return
	}
}

func NewTransactionList(outFileName string) *TransactionList {
	t := &TransactionList{
		outFileName: outFileName,
	}
	t.Load()

	return t
}
