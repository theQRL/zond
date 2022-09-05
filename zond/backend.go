package zond

import (
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/consensus"
	"github.com/theQRL/zond/internal/zondapi"
	"github.com/theQRL/zond/node"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/rpc"
)

type Zond struct {
	pos        *consensus.POS
	blockchain *chain.Chain
	APIBackend *ZondAPIBackend
}

func (s *Zond) APIs() []rpc.API {
	apis := zondapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	//apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return apis
}

func (s *Zond) BlockChain() *chain.Chain {
	return s.blockchain
}

func New(stack *node.Node, pos *consensus.POS) (*Zond, error) {
	z := &Zond{
		pos:        pos,
		blockchain: stack.Blockchain(),
	}
	z.APIBackend = &ZondAPIBackend{stack.Config().ExtRPCEnabled(),
		stack.Config().AllowUnprotectedTxs, z, ntp.GetNTP()}
	stack.RegisterAPIs(z.APIs())
	//z.shutdownTracker.MarkStartup()
	return z, nil
}
