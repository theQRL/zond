package view

import (
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
)

type PlainAddressState struct {
	Address string `json:"address" bson:"address"`
	Balance uint64 `json:"balance" bson:"balance"`
	Nonce   uint64 `json:"nonce" bson:"nonce"`
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
	a.Address = misc.BytesToHexStr(a2.Address)
	a.Balance = a2.Balance
	a.Nonce = a2.Nonce
}

func (a *PlainAddressState) FromData(address common.Address, balance uint64, nonce uint64) {
	a.Address = misc.BytesToHexStr(address[:])
	a.Balance = balance
	a.Nonce = nonce
}
