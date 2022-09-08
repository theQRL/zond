package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	common2 "github.com/theQRL/go-qrllib/common"
	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-qrllib/xmss"
	"github.com/theQRL/zond/api"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type Wallet struct {
	outFileName string

	pbData *protos.Wallet
}

func (w *Wallet) AddXMSS(height uint8, hashFunction xmss.HashFunction) {
	x := xmss.NewXMSSFromHeight(height, hashFunction)
	address := x.GetAddress()
	info := &protos.Info{
		Address:  misc.BytesToHexStr(address[:]),
		HexSeed:  x.GetHexSeed(),
		Mnemonic: x.GetMnemonic(),
		Type:     uint32(common2.XMSSSig),
	}
	w.pbData.Info = append(w.pbData.Info, info)

	fmt.Println("Added New XMSS Address: ", info.Address)

	w.Save()
}

func (w *Wallet) RecoverXMSSFromHexSeed(hexSeed [common2.ExtendedSeedSize]uint8) {
	x := xmss.NewXMSSFromExtendedSeed(hexSeed)
	address := x.GetAddress()
	info := &protos.Info{
		Address:  misc.BytesToHexStr(address[:]),
		HexSeed:  x.GetHexSeed(),
		Mnemonic: x.GetMnemonic(),
		Type:     uint32(common2.XMSSSig),
	}
	w.pbData.Info = append(w.pbData.Info, info)

	fmt.Println("Added Recovered XMSS Address: ", info.Address)

	w.Save()
}

func (w *Wallet) AddDilithium() {
	d := dilithium.New()
	address := d.GetAddress()
	info := &protos.Info{
		Address:  misc.BytesToHexStr(address[:]),
		HexSeed:  d.GetHexSeed(),
		Mnemonic: d.GetMnemonic(),
		Type:     uint32(common2.DilithiumSig),
	}
	w.pbData.Info = append(w.pbData.Info, info)

	fmt.Println("Added New Dilithium Address: ", info.Address)

	w.Save()
}

func (w *Wallet) RecoverDilithiumFromSeed(seed [common2.SeedSize]uint8) {
	d := dilithium.NewDilithiumFromSeed(seed)
	address := d.GetAddress()
	info := &protos.Info{
		Address:  misc.BytesToHexStr(address[:]),
		HexSeed:  d.GetHexSeed(),
		Mnemonic: d.GetMnemonic(),
		Type:     uint32(common2.DilithiumSig),
	}
	w.pbData.Info = append(w.pbData.Info, info)

	fmt.Println("Added Recovered Dilithium Address: ", info.Address)

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

	for {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		var response api.Response
		err = json.Unmarshal(bodyBytes, &response)

		// if response.Data is nil, then sleep for 6 seconds, as we hit the rate limit
		if response.Data == nil {
			time.Sleep(6 * time.Second)
			continue
		}

		balance := response.Data.(map[string]interface{})
		return strconv.ParseUint(balance["balance"].(string), 10, 64)
	}
}

func (w *Wallet) List() {
	for i, info := range w.pbData.Info {
		balance, err := w.reqBalance(info.Address)
		outputBalance := fmt.Sprintf("%d", balance)
		if err != nil {
			outputBalance = "?"
		}
		fmt.Println(fmt.Sprintf("%d\t%s\t%s",
			i+1, info.Address, outputBalance))
	}
}

func (w *Wallet) Secret() {
	for i, info := range w.pbData.Info {
		fmt.Println(fmt.Sprintf("%d\t%s\t%s\t\t%s",
			i+1, info.Address, info.HexSeed,
			info.Mnemonic))
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

func (w *Wallet) GetXMSSAccountByIndex(index uint) (*xmss.XMSS, error) {
	if int(index) > len(w.pbData.Info) {
		return nil, errors.New(fmt.Sprintf("Invalid Wallet Index"))
	}

	a := w.pbData.Info[index-1]
	if a.Type == uint32(common2.XMSSSig) {
		strHexSeed := a.HexSeed
		binHexSeed, err := misc.HexStrToBytes(strHexSeed)
		var binHexSeedSized [common2.ExtendedSeedSize]uint8
		copy(binHexSeedSized[:], binHexSeed)
		if err != nil {
			return nil, err
		}
		return xmss.NewXMSSFromExtendedSeed(binHexSeedSized), nil
	}

	return nil, fmt.Errorf("not an xmss account")
}

func (w *Wallet) GetDilithiumAccountByIndex(index uint) (*dilithium.Dilithium, error) {
	if index == 0 || int(index) > len(w.pbData.Info) {
		return nil, errors.New(fmt.Sprintf("Invalid Wallet Index"))
	}

	a := w.pbData.Info[index-1]
	if a.Type == uint32(common2.DilithiumSig) {
		strHexSeed := a.HexSeed
		binHexSeed, err := misc.HexStrToBytes(strHexSeed)
		var binHexSeedSized [common2.SeedSize]uint8
		copy(binHexSeedSized[:], binHexSeed)
		if err != nil {
			return nil, err
		}
		return dilithium.NewDilithiumFromSeed(binHexSeedSized), nil
	}

	return nil, fmt.Errorf("not a dilithium account")
}

func NewWallet(walletFileName string) *Wallet {
	w := &Wallet{
		outFileName: walletFileName,
	}
	w.Load()

	return w
}
