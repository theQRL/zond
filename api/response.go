package api

import (
	"github.com/theQRL/zond/api/view"
	"github.com/theQRL/zond/misc"
	"strconv"
)

type BroadcastTransactionResponse struct {
	TransactionHash string `json:"transactionHash"`
}

type AddressStateResponse struct {
	Address      string                `json:"address" bson:"address"`
	Balance      string                `json:"balance" bson:"balance"`
	Nonce        string                `json:"nonce" bson:"nonce"`
}

func NewAddressStateResponse(a *view.PlainAddressState) (*AddressStateResponse, error) {
	ar := &AddressStateResponse{
		Address: a.Address,
		Balance: misc.ShorToQuanta(a.Balance),
		Nonce: strconv.FormatUint(a.Nonce, 10),
	}
	return ar, nil
}
