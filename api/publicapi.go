package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/theQRL/zond/api/view"
	"github.com/theQRL/zond/chain"
	"github.com/theQRL/zond/common"
	"github.com/theQRL/zond/config"
	"github.com/theQRL/zond/misc"
	"github.com/theQRL/zond/ntp"
	"github.com/theQRL/zond/p2p/messages"
	"github.com/theQRL/zond/protos"
	"net/http"
	"strconv"
	"time"
)

type GetHeightResponse struct {
	Height uint64 `json:"height"`
}

type GetEstimatedNetworkFeeResponse struct {
	Fee string `json:"fee"`
}

type GetVersionResponse struct {
	Version string `json:"version"`
}

type Response struct {
	Error        uint        `json:"error"`
	ErrorMessage string      `json:"errorMessage"`
	Data         interface{} `json:"data"`
}

type PublicAPIServer struct {
	chain                    *chain.Chain
	ntp                      ntp.NTPInterface
	config                   *config.Config
	visitors                 *visitors
	registerAndBroadcastChan chan *messages.RegisterMessage
}

func (p *PublicAPIServer) Start() {
	c := config.GetConfig()

	router := mux.NewRouter()
	router.HandleFunc("/api/", p.RedirectToAPIDoc).Methods("GET")
	router.HandleFunc("/api/version", p.GetVersion).Methods("GET")
	router.HandleFunc("/api/block/{hash}", p.GetBlockByHash).Methods("GET")
	router.HandleFunc("/api/block/last", p.GetLastBlock).Methods("GET")
	router.HandleFunc("/api/address/{address}", p.GetAddressState).Methods("GET")
	router.HandleFunc("/api/balance/{address}", p.GetBalance).Methods("GET")
	router.HandleFunc("/api/fee", p.GetEstimatedNetworkFee).Methods("GET")
	router.HandleFunc("/api/broadcast/transfer", p.BroadcastTransferTx).Methods("POST")
	router.HandleFunc("/api/broadcast/stake", p.BroadcastStakeTx).Methods("POST")
	router.HandleFunc("/api/height", p.GetHeight).Methods("GET")
	router.HandleFunc("/api/evmcall", p.EVMCall).Methods("POST")
	//handler := cors.Default().Handler(router)
	co := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // service is available and allowed for this base url
		AllowedMethods: []string{
			http.MethodGet, //http methods for your app
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
			"post",
			"*",
			"*/*",
		},

		AllowedHeaders: []string{
			"*", //or you can your header key values which you are using in your application

		},
	})
	router.StrictSlash(false)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", c.User.API.PublicAPI.Host, c.User.API.PublicAPI.Port), co.Handler(router)))
}

func (p *PublicAPIServer) prepareResponse(errorCode uint, errorMessage string, data interface{}) *Response {
	r := &Response{
		Error:        errorCode,
		ErrorMessage: errorMessage,
		Data:         data,
	}
	return r
}

func (p *PublicAPIServer) RedirectToAPIDoc(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	http.Redirect(w, r, "https://api.theqrl.org/", 301)
}

func (p *PublicAPIServer) GetVersion(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	fmt.Println("Get version called")
	getVersionResponse := &GetVersionResponse{
		Version: p.config.Dev.Version,
	}
	json.NewEncoder(w).Encode(p.prepareResponse(0, "", getVersionResponse))
}

func (p *PublicAPIServer) GetBlockByHash(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	vars := mux.Vars(r)
	hash, found := vars["hash"]
	if !found {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			"block hash not provided",
			nil))
		return
	}
	binData, err := misc.HexStrToBytes(hash)
	var headerHash common.Hash
	copy(headerHash[:], binData)

	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error Decoding headerHash\n %s", err.Error()),
			nil))
		return
	}
	b, err := p.chain.GetBlock(headerHash)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error in GetBlock for headerHash %s\n %s", hash, err.Error()),
			nil))
		return
	}
	response := &view.PlainBlock{}
	response.BlockFromPBData(b.PBData(), b.Hash())
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) GetLastBlock(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	b := p.chain.GetLastBlock()

	response := &view.PlainBlock{}
	response.BlockFromPBData(b.PBData(), b.Hash())
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) GetAddressState(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	vars := mux.Vars(r)
	address := vars["address"]
	binData, err := misc.HexStrToBytes(address)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error Decoding address %s\n %s", address, err.Error()),
			nil))
		return
	}

	var binAddress common.Address
	copy(binAddress[:], binData)

	statedb, err := p.chain.AccountDB()
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Failed to get statedb %s\n %s", address, err.Error()),
			nil))
		return
	}

	balance := statedb.GetBalance(binAddress).Uint64()
	nonce := statedb.GetNonce(binAddress)

	response := &view.PlainAddressState{}
	response.FromData(binAddress, balance, nonce)

	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) GetBalance(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	vars := mux.Vars(r)
	address := vars["address"]
	binData, err := misc.HexStrToBytes(address)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error Decoding address %s\n %s", address, err.Error()),
			nil))
		return
	}

	var binAddress common.Address
	copy(binAddress[:], binData)

	statedb, err := p.chain.AccountDB()
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Failed to get statedb %s\n %s", address, err.Error()),
			nil))
		return
	}

	balance := statedb.GetBalance(binAddress).Uint64()

	response := view.PlainBalance{}
	response.Balance = strconv.FormatUint(balance, 10)
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) GetHeight(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	response := &GetHeightResponse{Height: p.chain.Height()}
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) GetNetworkStats(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
}

func (p *PublicAPIServer) GetEstimatedNetworkFee(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	// TODO: Fee needs to be calcuated by mean, median or mode
	response := &GetEstimatedNetworkFeeResponse{Fee: "1"}
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) BroadcastStakeTx(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var plainStakeTransaction view.PlainStakeTransaction
	err := decoder.Decode(&plainStakeTransaction)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error Decoding PlainStakeTransaction \n%s", err.Error()),
			nil))
		return
	}
	tx, err := plainStakeTransaction.ToStakeTransactionObject()
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error parsing ToStakeTransactionObject\n %s", err.Error()),
			nil))
		return
	}
	if err := p.chain.ValidateTransaction(tx.PBData()); err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			err.Error(),
			nil))
		return
	}

	txHash := tx.Hash()
	err = p.chain.GetTransactionPool().Add(
		tx,
		txHash,
		p.chain.GetLastBlock().SlotNumber(),
		p.ntp.Time())
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Failed to Add Txn into txn pool \n %s", err.Error()),
			nil))
		return
	}

	registerMessage := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			Data: &protos.LegacyMessage_StData{
				StData: tx.PBData(),
			},
			FuncName: protos.LegacyMessage_ST,
		},
		MsgHash: misc.BytesToHexStr(txHash[:]),
	}
	select {
	case p.registerAndBroadcastChan <- registerMessage:
	case <-time.After(10 * time.Second):
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			"Transaction Broadcast Timeout",
			nil))
		return
	}

	response := &BroadcastTransactionResponse{
		TransactionHash: misc.BytesToHexStr(txHash[:]),
	}
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) BroadcastTransferTx(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var plainTransferTransaction view.PlainTransferTransaction
	err := decoder.Decode(&plainTransferTransaction)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error Decoding PlainTransferTransaction \n%s", err.Error()),
			nil))
		return
	}
	tx, err := plainTransferTransaction.ToTransferTransactionObject()
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error parsing ToTransferTransactionObject\n %s", err.Error()),
			nil))
		return
	}

	if err := p.chain.ValidateTransaction(tx.PBData()); err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			err.Error(),
			nil))
		return
	}

	txHash := tx.Hash()
	err = p.chain.GetTransactionPool().Add(
		tx,
		txHash,
		p.chain.GetLastBlock().SlotNumber(),
		p.ntp.Time())
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Failed to Add Txn into txn pool \n %s", err.Error()),
			nil))
		return
	}

	registerMessage := &messages.RegisterMessage{
		Msg: &protos.LegacyMessage{
			Data: &protos.LegacyMessage_TtData{
				TtData: tx.PBData(),
			},
			FuncName: protos.LegacyMessage_TT,
		},
		MsgHash: misc.BytesToHexStr(txHash[:]),
	}
	select {
	case p.registerAndBroadcastChan <- registerMessage:
	case <-time.After(10 * time.Second):
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			"Transaction Broadcast Timeout",
			nil))
		return
	}
	response := &BroadcastTransactionResponse{
		TransactionHash: misc.BytesToHexStr(txHash[:]),
	}
	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func (p *PublicAPIServer) EVMCall(w http.ResponseWriter, r *http.Request) {
	// Check Rate Limit
	if !p.visitors.isAllowed(r.RemoteAddr) {
		w.WriteHeader(429)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var evmCall view.EVMCall
	err := decoder.Decode(&evmCall)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error Decoding EVMCall \n%s", err.Error()),
			nil))
		return
	}

	output, err := misc.HexStrToBytes(evmCall.Address)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error while decoding address \n%s", err.Error()),
			nil))
		return
	}

	var address common.Address
	copy(address[:], output)

	dataOutput, err := misc.HexStrToBytes(evmCall.Data)
	if err != nil {
		json.NewEncoder(w).Encode(p.prepareResponse(1,
			fmt.Sprintf("Error while decoding data \n%s", err.Error()),
			nil))
		return
	}

	result, err := p.chain.EVMCall(address, dataOutput)

	var response *EVMCallResponse
	if err != nil {
		response = &EVMCallResponse{
			Result: err.Error(),
		}
	} else {
		response = &EVMCallResponse{
			Result: misc.BytesToHexStr(result),
		}
	}

	json.NewEncoder(w).Encode(p.prepareResponse(0,
		"",
		response))
}

func NewPublicAPIServer(c *chain.Chain, registerAndBroadcastChan chan *messages.RegisterMessage) *PublicAPIServer {
	return &PublicAPIServer{
		chain:                    c,
		ntp:                      ntp.GetNTP(),
		config:                   config.GetConfig(),
		visitors:                 newVisitors(),
		registerAndBroadcastChan: registerAndBroadcastChan,
	}
}
