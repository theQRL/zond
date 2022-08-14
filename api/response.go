package api

type BroadcastTransactionResponse struct {
	TransactionHash string `json:"transactionHash"`
}

type EVMCallResponse struct {
	Result string `json:"result"`
}
