package messages

import (
	"github.com/theQRL/zond/protos"
)

type RegisterMessage struct {
	MsgHash string
	Msg     *protos.LegacyMessage
}
