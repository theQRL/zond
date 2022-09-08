package errmsg

const (
	TXInsufficientBalance          = "[%s] insufficient balance | txhash %s | addr %s | required balance %d | balance available %d"
	TXInsufficientGas              = "[%s] insufficient gas | txhash %s | addr %s | gas provided %d | gas required %d"
	TXInvalidTxHash                = "[%s] invalid tx hash %s | addr %s | expected tx hash %s"
	TXInvalidNonce                 = "[%s] invalid nonce | txhash %s | addr %s | found nonce %d | expected nonce %d"
	TXInvalidStakeAmount           = "[%s] invalid stake amount | txhash %s | addr %s | stake amount %d | minimum required stake amount %d"
	TXInvalidAddrFrom              = "[%s] invalid from address | txhash %s | addr %s"
	TXInvalidAddrTo                = "[%s] invalid to address | txhash %s | addr %s"
	TXAddrFromCannotBeCoinBaseAddr = "[%s] from address cannot be a coinbase address | txhash %s | addr %s"
	TXAddrToCannotBeCoinBaseAddr   = "[%s] to address cannot be a coinbase address | txhash %s | addr %s"
	TXDataLengthExceedsLimit       = "[%s] data length exceeds limit | txhash %s | addr %s | data len %d | limit %d"

	TXDilithiumSignatureVerificationFailed = "[%s] dilithium signature verification failed | txhash %s | addr %s"
	TXInvalidDilithiumAddrFrom             = "[%s] invalid dilithium address | txhash %s | addr_from %s"
	TXInvalidDilithiumPKSize               = "[%s] invalid dilithium pk | txhash %s | addr %s | pk size %d"

	TXXMSSSignatureVerificationFailed = "[%s] xmss signature verification failed | txhash %s | addr %s"
	TXInvalidXMSSOTSIndex             = "[%s] invalid xmss ots index | txhash %s | addr %s | ots index %d is already used"

	TXInvalidAttestor   = "[%s] invalid attestor | txhash %s | addr %s is not an attestor for slot number #%d"
	TXInvalidSlotLeader = "[%s] invalid slot leader/block proposer | txhash %s | addr %s is not a slot leader/block proposer for slot number #%d | expected slot leader %s"

	TXInvalidBlockReward    = "[%s] invalid block reward | txhash %s | addr %s | block reward %d | expected block reward %d"
	TXInvalidAttestorReward = "[%s] invalid attestor reward | txhash %s | addr %s | attestor reward %d | expected attestor reward %d"

	TXInsufficientStakeBalance = "[%s] insufficient stake balance | txhash %s | addr %s | stake balance %d"
)
