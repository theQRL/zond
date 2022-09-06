package zondapi

import (
	"context"
	"errors"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/common/hexutil"
	"github.com/theQRL/zond/common/math"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/log"
	"math/big"
)

// TransactionArgs represents the arguments to construct a new transaction
// or a message call.
type TransactionArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	Data *hexutil.Bytes `json:"data"`

	// Introduced by AccessListTxType transaction.
	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`
}

// from retrieves the transaction sender address.
func (args *TransactionArgs) from() common.Address {
	if args.From == nil {
		return common.Address{}
	}
	return *args.From
}

// setDefaults fills in default values for unspecified tx fields.
func (args *TransactionArgs) setDefaults(ctx context.Context, b Backend) error {
	//if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
	//	return errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	//}
	//// After london, default to 1559 unless gasPrice is set
	//head := b.CurrentHeader()
	//// If user specifies both maxPriorityfee and maxFee, then we do not
	//// need to consult the chain for defaults. It's definitely a London tx.
	//if args.MaxPriorityFeePerGas == nil || args.MaxFeePerGas == nil {
	//	// In this clause, user left some fields unspecified.
	//	if b.ChainConfig().IsLondon(head.Number) && args.GasPrice == nil {
	//		if args.MaxPriorityFeePerGas == nil {
	//			tip, err := b.SuggestGasTipCap(ctx)
	//			if err != nil {
	//				return err
	//			}
	//			args.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
	//		}
	//		if args.MaxFeePerGas == nil {
	//			gasFeeCap := new(big.Int).Add(
	//				(*big.Int)(args.MaxPriorityFeePerGas),
	//				new(big.Int).Mul(head.BaseFee, big.NewInt(2)),
	//			)
	//			args.MaxFeePerGas = (*hexutil.Big)(gasFeeCap)
	//		}
	//		if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
	//			return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
	//		}
	//	} else {
	//		if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
	//			return errors.New("maxFeePerGas or maxPriorityFeePerGas specified but london is not active yet")
	//		}
	//		if args.GasPrice == nil {
	//			price, err := b.SuggestGasTipCap(ctx)
	//			if err != nil {
	//				return err
	//			}
	//			if b.ChainConfig().IsLondon(head.Number) {
	//				// The legacy tx gas price suggestion should not add 2x base fee
	//				// because all fees are consumed, so it would result in a spiral
	//				// upwards.
	//				price.Add(price, head.BaseFee)
	//			}
	//			args.GasPrice = (*hexutil.Big)(price)
	//		}
	//	}
	//} else {
	//	// Both maxPriorityfee and maxFee set by caller. Sanity-check their internal relation
	//	if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
	//		return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
	//	}
	//}
	//if args.Value == nil {
	//	args.Value = new(hexutil.Big)
	//}
	//if args.Nonce == nil {
	//	nonce, err := b.GetPoolNonce(ctx, args.from())
	//	if err != nil {
	//		return err
	//	}
	//	args.Nonce = (*hexutil.Uint64)(&nonce)
	//}
	//if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
	//	return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	//}
	//if args.To == nil && len(args.data()) == 0 {
	//	return errors.New(`contract creation without any data provided`)
	//}
	//// Estimate the gas usage if necessary.
	//if args.Gas == nil {
	//	// These fields are immutable during the estimation, safe to
	//	// pass the pointer directly.
	//	data := args.data()
	//	callArgs := TransactionArgs{
	//		From:                 args.From,
	//		To:                   args.To,
	//		GasPrice:             args.GasPrice,
	//		MaxFeePerGas:         args.MaxFeePerGas,
	//		MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
	//		Value:                args.Value,
	//		Data:                 (*hexutil.Bytes)(&data),
	//		AccessList:           args.AccessList,
	//	}
	//	pendingBlockNr := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
	//	estimated, err := DoEstimateGas(ctx, b, callArgs, pendingBlockNr, b.RPCGasCap())
	//	if err != nil {
	//		return err
	//	}
	//	args.Gas = &estimated
	//	log.Trace("Estimate gas usage automatically", "gas", args.Gas)
	//}
	//if args.ChainID == nil {
	//	id := (*hexutil.Big)(b.ChainConfig().ChainID)
	//	args.ChainID = id
	//}
	return nil
}

// ToMessage converts the transaction arguments to the Message type used by the
// core evm. This method is used in calls and traces that do not require a real
// live transaction.
func (args *TransactionArgs) ToMessage(globalGasCap uint64, baseFee *big.Int) (types.Message, error) {
	// Reject invalid combinations of pre- and post-1559 fee styles
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return types.Message{}, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	// Set sender address or use zero address if none specified.
	addr := args.from()

	// Set default gas & gas price if none were set
	gas := globalGasCap
	if gas == 0 {
		gas = uint64(math.MaxUint64 / 2)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != 0 && globalGasCap < gas {
		log.Warn("Caller gas above allowance, capping", "requested", gas, "cap", globalGasCap)
		gas = globalGasCap
	}
	var (
		gasPrice  *big.Int
		gasFeeCap *big.Int
		gasTipCap *big.Int
	)
	if baseFee == nil {
		// If there's no basefee, then it must be a non-1559 execution
		gasPrice = new(big.Int)
		if args.GasPrice != nil {
			gasPrice = args.GasPrice.ToInt()
		}
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else {
		// A basefee is provided, necessitating 1559-type execution
		if args.GasPrice != nil {
			// User specified the legacy gas field, convert to 1559 gas typing
			gasPrice = args.GasPrice.ToInt()
			gasFeeCap, gasTipCap = gasPrice, gasPrice
		} else {
			// User specified 1559 gas feilds (or none), use those
			gasFeeCap = new(big.Int)
			if args.MaxFeePerGas != nil {
				gasFeeCap = args.MaxFeePerGas.ToInt()
			}
			gasTipCap = new(big.Int)
			if args.MaxPriorityFeePerGas != nil {
				gasTipCap = args.MaxPriorityFeePerGas.ToInt()
			}
			// Backfill the legacy gasPrice for EVM execution, unless we're all zeroes
			gasPrice = new(big.Int)
			if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
				gasPrice = math.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
			}
		}
	}
	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}
	data := *args.Data
	var accessList types.AccessList
	if args.AccessList != nil {
		accessList = *args.AccessList
	}
	msg := types.NewMessage(addr, args.To, 0, value, gas, gasPrice, gasFeeCap, gasTipCap, data, accessList, true)
	return msg, nil
}

// toTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) toTransaction() *types.Transaction {
	var data types.TxData
	switch {
	case args.MaxFeePerGas != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.DynamicFeeTx{
			To:         args.To,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(args.Value),
			Data:       *args.Data,
			AccessList: al,
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			To:         args.To,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       *args.Data,
			AccessList: *args.AccessList,
		}
	default:
		data = &types.LegacyTx{
			To:       args.To,
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     *args.Data,
		}
	}
	return types.NewTx(data)
}

// ToTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) ToTransaction() *types.Transaction {
	return args.toTransaction()
}
