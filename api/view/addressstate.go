package view

import (
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type PlainAddressState struct {
	Address           string   `json:"address" bson:"address"`
	Balance           uint64   `json:"balance" bson:"balance"`
	Nonce             uint64   `json:"nonce" bson:"nonce"`
	//OtsBitfield       []string `json:"otsBitField" bson:"otsBitField"`
}

type PlainBalance struct {
	Balance string `json:"balance" bson:"balance"`
}

type NextUnusedOTS struct {
	UnusedOTSIndex uint64 `json:"unusedOtsIndex" bson:"unusedOtsIndex"`
	Found          bool   `json:"found" bson:"found"`
}

type IsUnusedOTSIndex struct {
	IsUnusedOTSIndex bool `json:"isUnusedOtsIndex" bson:"isUnusedOtsIndex"`
}

func (a *PlainAddressState) AddressStateFromPBData(a2 *protos.AddressState) {
	a.Address = misc.Bin2Qaddress(a2.Address)
	a.Balance = a2.Balance
	a.Nonce = a2.Nonce
	//for i := 0; i < int(config.GetConfig().Dev.OtsBitFieldSize); i++ {
	//	a.OtsBitfield = append(a.OtsBitfield, base64.StdEncoding.EncodeToString(a2.OtsBitfield[i]))
	//}
}