package p2p

import (
	"github.com/theQRL/zond/protos"
	"time"
)

type Msg struct {
	msg        *protos.LegacyMessage
	ReceivedAt time.Time
}
