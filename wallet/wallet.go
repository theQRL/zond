package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/crypto"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Wallet struct {
	outFileName string

	pbData *protos.Wallet
}

func (w *Wallet) Add(height uint64, hashFunction goqrllib.EHashFunction) {
	xmss := crypto.FromHeight(height, hashFunction)
	xmssInfo := &protos.XMSSInfo{
		Address: xmss.StrAddress(),
		HexSeed: xmss.HexSeed(),
		Mnemonic: xmss.Mnemonic(),
	}
	w.pbData.XmssInfo = append(w.pbData.XmssInfo, xmssInfo)

	fmt.Println("Added New Zond Address: ", xmssInfo.Address)

	w.Save()
}

func (w *Wallet) reqBalance(address string) (uint64, error) {
	userConfig := config.GetUserConfig()
	apiHostPort := fmt.Sprintf("http://%s:%d",
		userConfig.API.PublicAPI.Host,
		userConfig.API.PublicAPI.Port)
	url := fmt.Sprintf("%s/api/balance/%s", apiHostPort, address)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	var response api.Response
	err = json.Unmarshal(bodyBytes, &response)

	balance := response.Data.(map[string]interface{})
	return strconv.ParseUint(balance["balance"].(string), 10, 64)
}

func (w *Wallet) List() {
	for i, xmssInfo := range w.pbData.XmssInfo {
		balance, err := w.reqBalance(xmssInfo.Address)
		outputBalance := fmt.Sprintf("%d", balance)
		if err != nil {
			outputBalance = "?"
		}
		fmt.Println(fmt.Sprintf("%d\t%s\t%s",
			i + 1, xmssInfo.Address, outputBalance))
	}
}

func (w *Wallet) Secret() {
	for i, xmssInfo := range w.pbData.XmssInfo {
		fmt.Println(fmt.Sprintf("%d\t%s\t%s\t%s",
			i + 1, xmssInfo.Address, xmssInfo.HexSeed,
			xmssInfo.Mnemonic))
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

	if !misc.FileExists(w.outFileName) {
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

func (w *Wallet) GetXMSSByIndex(index uint) (*crypto.XMSS, error) {
	if int(index) > len(w.pbData.XmssInfo) {
		return nil, errors.New(fmt.Sprintf("Invalid XMSS Index"))
	}
	strHexSeed := w.pbData.XmssInfo[index - 1].HexSeed
	return crypto.FromExtendedSeed(misc.HStr2Bin(strHexSeed)), nil
}

func NewWallet(walletFileName string) *Wallet {
	w := &Wallet{
		outFileName: walletFileName,
	}
	w.Load()

	return w
}
