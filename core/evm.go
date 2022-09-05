// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"crypto/sha256"
	"github.com/theQRL/zond/block"
	"math/big"

	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/core/vm"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	// Engine() consensus.Engine

	// GetBlockBySlotNumber returns the hash corresponding to their hash.
	GetBlockBySlotNumber(uint64) (*block.Block, error)
}

// NewEVMBlockContext creates a new context for use in the EVM.
func NewEVMBlockContext(header *block.Header, getHashFunc vm.GetHashFunc, author *common.Address) vm.BlockContext {
	var (
		beneficiary common.Address
		baseFee     *big.Int
		random      *common.Hash
	)

	beneficiary = *author

	if header.BaseFee() != nil {
		baseFee = new(big.Int).Set(header.BaseFee())
	}
	// TODO: Find the purpose of random
	//if header.Difficulty.Cmp(common.Big0) == 0 {
	//	random = &header.MixDigest
	//}
	// TODO: to be removed when we have found the use of random and if it is even needed?
	h := sha256.New()
	ph := header.ParentHash()
	h.Write(ph[:])
	hashedOutput := h.Sum(nil)
	var randomOutput common.Hash
	copy(randomOutput[:], hashedOutput)
	random = &randomOutput

	return vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		//GetHash:     GetHashFn(header, chain),
		GetHash:     getHashFunc,
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).Set(header.Number()),
		Time:        new(big.Int).SetUint64(header.Timestamp().Uint64()),
		//Difficulty:  new(big.Int).Set(header.Difficulty),
		Difficulty: new(big.Int), // Difficulty set to 0
		BaseFee:    baseFee,
		GasLimit:   header.GasLimit(),
		Random:     random,
	}
}

// NewEVMTxContext creates a new transaction context for a single transaction.
func NewEVMTxContext(msg Message) vm.TxContext {
	return vm.TxContext{
		Origin:   msg.From(),
		GasPrice: new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
//func GetHashFn(ref *block.Header, chain ChainContext) func(n uint64) common.Hash {
//	// Cache will initially contain [refHash.parent],
//	// Then fill up with [refHash.p, refHash.pp, refHash.ppp, ...]
//	var cache []common.Hash
//
//	return func(n uint64) common.Hash {
//		// If there's no hash cache yet, make one
//		if len(cache) == 0 {
//			cache = append(cache, ref.ParentHash())
//		}
//		if idx := ref.Number().Uint64() - n - 1; idx < uint64(len(cache)) {
//			return cache[idx]
//		}
//		// No luck in the cache, but we can start iterating from the last element we already know
//		lastKnownHash := cache[len(cache)-1]
//		lastKnownNumber := ref.Number().Uint64() - uint64(len(cache))
//
//		for {
//			header := chain.GetHeader(lastKnownHash, lastKnownNumber)
//			if header == nil {
//				break
//			}
//			cache = append(cache, header.ParentHash())
//			lastKnownHash = header.ParentHash()
//			lastKnownNumber = header.Number().Uint64() - 1
//			if n == lastKnownNumber {
//				return lastKnownHash
//			}
//		}
//		return common.Hash{}
//	}
//}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
