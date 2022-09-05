package p2p

import (
	"github.com/deckarep/golang-set"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/protos"
	"sync"
)

type OrderedMap struct {
	mapping map[string]*MessageRequest
	order   []string
}

func (o *OrderedMap) Put(k string, v *MessageRequest) {
	o.mapping[k] = v
	o.order = append(o.order, k)
}

func (o *OrderedMap) Get(k string) *MessageRequest {
	return o.mapping[k]
}

func (o *OrderedMap) Delete(k string) {
	delete(o.mapping, k)
	o.order = o.order[1:] // TODO: Need Optimization
}

type MessageReceipt struct {
	lock sync.Mutex

	allowedTypes mapset.Set
	servicesArgs *map[protos.LegacyMessage_FuncName]string

	hashMsg      map[string]*protos.LegacyMessage
	hashMsgOrder []string

	requestedHash      map[string]*MessageRequest
	requestedHashOrder []string

	c *config.Config
}

func (mr *MessageReceipt) addPeer(mrData *protos.MRData, peer *Peer) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	msgType := mrData.Type
	msgHash := misc.BytesToHexStr(mrData.Hash)
	if !mr.allowedTypes.Contains(msgType) {
		log.Error("Message Type ", msgType, " not found in allowed types")
		return
	}

	if len(mr.requestedHashOrder) >= int(mr.c.Dev.MessageQSize) {
		delete(mr.requestedHash, mr.requestedHashOrder[0])
		mr.requestedHashOrder = mr.requestedHashOrder[1:]
	}

	if _, ok := mr.requestedHash[msgHash]; !ok {
		mr.requestedHash[msgHash] = CreateMessageRequest(mrData, peer)
		mr.requestedHashOrder = append(mr.requestedHashOrder, msgHash)
	} else {
		mr.requestedHash[msgHash].addPeer(peer)
	}
}

func (mr *MessageReceipt) IsRequested(msgHashBytes common.Hash, peer *Peer) bool {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	msgHash := misc.BytesToHexStr(msgHashBytes[:])
	if requestedHash, ok := mr.requestedHash[msgHash]; ok {
		if _, ok := requestedHash.peers[peer]; ok {
			return true
		}
	}

	mr.removePeerFromRequestedHash(msgHash, peer)
	return false
}

func (mr *MessageReceipt) removePeerFromRequestedHash(msgHash string, peer *Peer) {
	if messageRequest, ok := mr.requestedHash[msgHash]; ok {
		if _, ok := messageRequest.peers[peer]; ok {
			delete(messageRequest.peers, peer)
			if len(messageRequest.peers) == 0 {
				mr.removeRequestedHash(msgHash)
			}
		}
	}
}

func (mr *MessageReceipt) removeRequestedHash(msgHash string) {
	delete(mr.requestedHash, msgHash)
	for index, hash := range mr.requestedHashOrder {
		if hash == msgHash {
			mr.requestedHashOrder = append(mr.requestedHashOrder[:index], mr.requestedHashOrder[index+1:]...)
			break
		}
	}
}

func (mr *MessageReceipt) RemoveRequestedHash(msgHash string) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	mr.removeRequestedHash(msgHash)
}

func (mr *MessageReceipt) contains(msgHashBytes []byte, messageType protos.LegacyMessage_FuncName) bool {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	msgHash := misc.BytesToHexStr(msgHashBytes)
	value, ok := mr.hashMsg[msgHash]
	if !ok {
		return false
	}

	return value.FuncName == messageType
}

func (mr *MessageReceipt) Get(messageHash []byte) *protos.LegacyMessage {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	msgHash := misc.BytesToHexStr(messageHash)
	value, _ := mr.hashMsg[msgHash]
	return value
}

func (mr *MessageReceipt) GetHashMsg(msgHash string) (value *protos.LegacyMessage, ok bool) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	value, ok = mr.hashMsg[msgHash]
	return
}

func (mr *MessageReceipt) GetRequestedHash(msgHash string) (value *MessageRequest, ok bool) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	value, ok = mr.requestedHash[msgHash]
	return
}

func (mr *MessageReceipt) Register(msgHash string, msg *protos.LegacyMessage) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	if uint32(len(mr.hashMsg)) >= mr.c.Dev.MessageQSize {
		delete(mr.hashMsg, mr.hashMsgOrder[0])
		mr.hashMsgOrder = mr.hashMsgOrder[1:]
	}

	mr.hashMsg[msgHash] = msg
	mr.hashMsgOrder = append(mr.hashMsgOrder, msgHash)
}

func CreateMR() (mr *MessageReceipt) {

	allowedTypes := mapset.NewSet()
	allowedTypes.Add(protos.LegacyMessage_BA)
	allowedTypes.Add(protos.LegacyMessage_BK)
	allowedTypes.Add(protos.LegacyMessage_TT)
	allowedTypes.Add(protos.LegacyMessage_ST)
	allowedTypes.Add(protos.LegacyMessage_AT)

	servicesArgs := &map[protos.LegacyMessage_FuncName]string{
		protos.LegacyMessage_VE:   "veData", // Version Data
		protos.LegacyMessage_PL:   "plData", // Peer List Data
		protos.LegacyMessage_PONG: "pongData",

		protos.LegacyMessage_MR:  "mrData", // Message Receipt Data
		protos.LegacyMessage_SFM: "mrData", // Message Response Data

		protos.LegacyMessage_BA: "baData", // Block For Attestation
		protos.LegacyMessage_BK: "block",
		protos.LegacyMessage_FB: "fbData", // Fetch Block Data
		protos.LegacyMessage_PB: "pbData", // Push Block Data

		protos.LegacyMessage_TT: "ttData", // Transfer Transaction
		protos.LegacyMessage_ST: "stData", // Stake Transaction
		protos.LegacyMessage_AT: "atData", // Attest Transaction

		protos.LegacyMessage_SYNC: "syncData",
	}

	mr = &MessageReceipt{
		allowedTypes:       allowedTypes,
		hashMsg:            make(map[string]*protos.LegacyMessage),
		hashMsgOrder:       make([]string, 1),
		requestedHash:      make(map[string]*MessageRequest),
		requestedHashOrder: make([]string, 1),
		servicesArgs:       servicesArgs,
		c:                  config.GetConfig(),
	}

	return
}
