package p2p

import "quantum-blockchain/blockchain"

// Базовые типы сообщений
const (
	MsgBlock    = "block"
	MsgTx       = "transaction"
	MsgGetBlock = "getblock"
)

// BlockMessage для передачи блоков
type BlockMessage struct {
	Block *blockchain.Block
}

// TxMessage для передачи транзакций
type TxMessage struct {
	Tx *blockchain.Transaction
}

// GetBlockMessage для запроса блоков
type GetBlockMessage struct {
	Hash []byte
}