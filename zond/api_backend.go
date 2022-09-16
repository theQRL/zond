package zond

import (
	"context"
	"errors"
	"github.com/theQRL/zond/block"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/core"
	"github.com/theQRL/zond/core/state"
	"github.com/theQRL/zond/core/types"
	"github.com/theQRL/zond/core/vm"
	"github.com/theQRL/zond/metadata"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/params"
	"github.com/theQRL/zond/protos"
	"github.com/theQRL/zond/rpc"
	"github.com/theQRL/zond/transactions"
	"time"
)

// ZondAPIBackend implements ethapi.Backend for full nodes
type ZondAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	zond                *Zond
	ntp                 ntp.NTPInterface
	//gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *ZondAPIBackend) ChainConfig() *params.ChainConfig {
	return b.zond.blockchain.Config()
}

//func (b *ZondAPIBackend) CurrentBlock() *types.Block {
//	return b.zond.blockchain.CurrentBlock()
//}
//
//func (b *ZondAPIBackend) SetHead(number uint64) {
//	b.zond.handler.downloader.Cancel()
//	b.zond.blockchain.SetHead(number)
//}

func (b *ZondAPIBackend) GetValidators(ctx context.Context) (*metadata.EpochMetaData, error) {
	return b.zond.blockchain.GetValidators()
}

func (b *ZondAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*protos.BlockHeader, error) {
	// Pending block is only known by the miner
	//if number == rpc.PendingBlockNumber {
	//	block := b.zond.miner.PendingBlock()
	//	return block.Header(), nil
	//}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.zond.blockchain.CurrentBlock().Header().PBData(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		return b.zond.blockchain.CurrentFinalizedBlock().Header().PBData(), nil
	}
	block := b.zond.blockchain.GetBlockByNumber(uint64(number))
	return block.Header().PBData(), nil
}

//func (b *ZondAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
//	if blockNr, ok := blockNrOrHash.Number(); ok {
//		return b.HeaderByNumber(ctx, blockNr)
//	}
//	if hash, ok := blockNrOrHash.Hash(); ok {
//		header := b.zond.blockchain.GetHeaderByHash(hash)
//		if header == nil {
//			return nil, errors.New("header for hash not found")
//		}
//		if blockNrOrHash.RequireCanonical && b.zond.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
//			return nil, errors.New("hash is not currently canonical")
//		}
//		return header, nil
//	}
//	return nil, errors.New("invalid arguments; neither block nor hash specified")
//}

func (b *ZondAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*protos.BlockHeader, error) {
	//return b.zond.blockchain.GetHeaderByHash(hash), nil
	block, err := b.zond.blockchain.GetBlock(hash)
	if err != nil {
		return nil, err
	}
	return block.Header().PBData(), nil
}

func (b *ZondAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*protos.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.zond.pos.PendingBlock()
		if block == nil {
			return nil, nil
		}
		return block.PBData(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.zond.blockchain.GetLastBlock().PBData(), nil
	}
	//if number == rpc.FinalizedBlockNumber {
	//	return b.zond.blockchain.CurrentFinalizedBlock(), nil
	//}
	block := b.zond.blockchain.GetBlockByNumber(uint64(number))
	if block == nil {
		return nil, nil
	}
	return block.PBData(), nil
}

func (b *ZondAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*protos.Block, error) {
	block, err := b.zond.blockchain.GetBlock(hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}
	return block.PBData(), nil
}

func (b *ZondAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*protos.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err := b.zond.blockchain.GetBlock(hash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block.PBData(), nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ZondAPIBackend) PendingBlockAndReceipts() (*protos.Block, types.Receipts) {
	//return b.zond.miner.PendingBlockAndReceipts()
	return nil, nil
}

func (b *ZondAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *protos.BlockHeader, error) {
	// Pending state is only known by the miner
	//if number == rpc.PendingBlockNumber {
	//	block, state := b.zond.miner.Pending()
	//	return state, block.Header(), nil
	//}
	// Otherwise resolve the block number and return its state
	if rpc.BlockNumber(b.zond.BlockChain().GetLastBlock().SlotNumber()) < number || number < 0 {
		number = rpc.BlockNumber(b.zond.BlockChain().GetLastBlock().SlotNumber())
	}
	block := b.zond.BlockChain().GetBlockByNumber(uint64(number))
	if block == nil {
		return nil, nil, errors.New("header not found")
	}

	bm, err := b.zond.BlockChain().GetBlockMetaData(block.Hash())
	if err != nil {
		return nil, nil, err
	}

	//stateDb, err := b.zond.BlockChain().StateAt(header.Root)
	stateDb, err := b.zond.BlockChain().StateAt(bm.TrieRoot())

	return stateDb, block.Header().PBData(), err
}

func (b *ZondAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *protos.BlockHeader, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err := b.zond.BlockChain().GetBlock(hash)
		if err != nil {
			return nil, nil, err
		}

		bm, err := b.zond.BlockChain().GetBlockMetaData(block.Hash())
		if err != nil {
			return nil, nil, err
		}

		// TODO (cyyber): This needs to be implemented and require some design changes
		//if blockNrOrHash.RequireCanonical && b.zond.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
		//	return nil, nil, errors.New("hash is not currently canonical")
		//}

		//stateDb, err := b.zond.BlockChain().StateAt(header.Root)
		stateDb, err := b.zond.BlockChain().StateAt(bm.TrieRoot())
		return stateDb, block.Header().PBData(), err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ZondAPIBackend) GetReceipts(ctx context.Context, hash common.Hash, isProtocolTransaction bool) (types.Receipts, error) {
	//return b.zond.blockchain.GetReceiptsByHash(hash), nil
	return b.zond.blockchain.GetReceiptsByHash(hash, isProtocolTransaction), nil
}

func (b *ZondAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	return b.zond.blockchain.GetLogsByHash(hash)
}

//func (b *ZondAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
//	if header := b.zond.blockchain.GetHeaderByHash(hash); header != nil {
//		return b.zond.blockchain.GetTd(hash, header.Number.Uint64())
//	}
//	return nil
//}

func (b *ZondAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *protos.BlockHeader, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.zond.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)

	// TODO (cyyber): Fix getHashFunc && author
	blockData, err := b.zond.BlockChain().GetBlock(common.BytesToHash(header.Hash))
	if err != nil {
		return nil, vmError, err
	}
	author := blockData.GetBlockProposer()
	context := core.NewEVMBlockContext(block.HeaderFromPBData(header), nil, &author)

	return vm.NewEVM(context, txContext, state, b.zond.blockchain.Config(), *vmConfig), vmError, nil
}

//func (b *ZondAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
//	return b.zond.BlockChain().SubscribeRemovedLogsEvent(ch)
//}
//
//func (b *ZondAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
//	return b.zond.miner.SubscribePendingLogs(ch)
//}
//
//func (b *ZondAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
//	return b.zond.BlockChain().SubscribeChainEvent(ch)
//}
//
//func (b *ZondAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
//	return b.zond.BlockChain().SubscribeChainHeadEvent(ch)
//}
//
//func (b *ZondAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
//	return b.zond.BlockChain().SubscribeChainSideEvent(ch)
//}
//
//func (b *ZondAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
//	return b.zond.BlockChain().SubscribeLogsEvent(ch)
//}

func (b *ZondAPIBackend) SendTx(ctx context.Context, signedTx transactions.TransactionInterface) error {
	if err := b.zond.blockchain.ValidateTransaction(signedTx.PBData()); err != nil {
		return err
	}
	return b.zond.blockchain.GetTransactionPool().Add(signedTx, signedTx.Hash(), b.zond.blockchain.GetLastBlock().SlotNumber(),
		b.ntp.Time())
}

//func (b *ZondAPIBackend) GetPoolTransactions() (types.Transactions, error) {
//	pending := b.zond.txPool.Pending(false)
//	var txs types.Transactions
//	for _, batch := range pending {
//		txs = append(txs, batch...)
//	}
//	return txs, nil
//}
//
//func (b *ZondAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
//	return b.zond.txPool.Get(hash)
//}

func (b *ZondAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*protos.Transaction, common.Hash, uint64, uint64, error) {
	//tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.zond.ChainDb(), txHash)
	//return tx, blockHash, blockNumber, index, nil
	tx, blockHash, blockNumber, index := b.zond.BlockChain().GetTransactionMetaDataByHash(txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *ZondAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	//return b.zond.txPool.Nonce(addr), nil
	return b.zond.blockchain.GetNonce(addr)
}

//func (b *ZondAPIBackend) Stats() (pending int, queued int) {
//	return b.zond.txPool.Stats()
//}
//
//func (b *ZondAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
//	return b.zond.TxPool().Content()
//}
//
//func (b *ZondAPIBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
//	return b.zond.TxPool().ContentFrom(addr)
//}
//
//func (b *ZondAPIBackend) TxPool() *core.TxPool {
//	return b.zond.TxPool()
//}
//
//func (b *ZondAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
//	return b.zond.TxPool().SubscribeNewTxsEvent(ch)
//}
//
//func (b *ZondAPIBackend) SyncProgress() ethereum.SyncProgress {
//	return b.zond.Downloader().Progress()
//}
//
//func (b *ZondAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
//	return b.gpo.SuggestTipCap(ctx)
//}
//
//func (b *ZondAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
//	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
//}
//
//func (b *ZondAPIBackend) ChainDb() ethdb.Database {
//	return b.zond.ChainDb()
//}
//
//func (b *ZondAPIBackend) EventMux() *event.TypeMux {
//	return b.zond.EventMux()
//}
//
//func (b *ZondAPIBackend) AccountManager() *accounts.Manager {
//	return b.zond.AccountManager()
//}
//
//func (b *ZondAPIBackend) ExtRPCEnabled() bool {
//	return b.extRPCEnabled
//}
//
//func (b *ZondAPIBackend) UnprotectedAllowed() bool {
//	return b.allowUnprotectedTxs
//}

func (b *ZondAPIBackend) RPCGasCap() uint64 {
	// TODO (cyyber): Add a separate config for RPCGasCap
	return config.GetDevConfig().BlockGasLimit
	//return b.zond.config.RPCGasCap
}

func (b *ZondAPIBackend) RPCEVMTimeout() time.Duration {
	// TODO (cyyber): Move this to config
	return 5 * time.Second
	//return b.zond.config.RPCEVMTimeout
}

//func (b *ZondAPIBackend) RPCTxFeeCap() float64 {
//	return b.zond.config.RPCTxFeeCap
//}
//
//func (b *ZondAPIBackend) BloomStatus() (uint64, uint64) {
//	sections, _, _ := b.zond.bloomIndexer.Sections()
//	return params.BloomBitsBlocks, sections
//}
//
//func (b *ZondAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
//	for i := 0; i < bloomFilterThreads; i++ {
//		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.zond.bloomRequests)
//	}
//}
//
//func (b *ZondAPIBackend) Engine() consensus.Engine {
//	return b.zond.engine
//}
//
//func (b *ZondAPIBackend) CurrentHeader() *types.Header {
//	return b.zond.blockchain.CurrentHeader()
//}
//
//func (b *ZondAPIBackend) Miner() *miner.Miner {
//	return b.zond.Miner()
//}
//
//func (b *ZondAPIBackend) StartMining(threads int) error {
//	return b.zond.StartMining(threads)
//}
//
//func (b *ZondAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, checkLive, preferDisk bool) (*state.StateDB, error) {
//	return b.zond.StateAtBlock(block, reexec, base, checkLive, preferDisk)
//}
//
//func (b *ZondAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
//	return b.zond.stateAtTransaction(block, txIndex, reexec)
//}
