package transactions

import "github.com/theQRL/zond/protos"

type TxType uint8

const (
	TypeCoinBase TxType = iota
	TypeAttest
	TypeTransfer
	TypeStake
)

func GetProtocolTransactionType(protoTX *protos.ProtocolTransaction) TxType {
	switch protoTX.Type.(type) {
	case *protos.ProtocolTransaction_CoinBase:
		return TypeCoinBase
	case *protos.ProtocolTransaction_Attest:
		return TypeAttest
	}
	panic("invalid protocol transaction type")
}

func GetTransactionType(protoTX *protos.Transaction) TxType {
	switch protoTX.Type.(type) {
	case *protos.Transaction_Transfer:
		return TypeTransfer
	case *protos.Transaction_Stake:
		return TypeStake
	}
	panic("invalid transaction type")
}
