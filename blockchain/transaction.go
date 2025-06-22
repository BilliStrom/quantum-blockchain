package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"log"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func CoinbaseTx(to []byte) *Transaction {
	txIn := TxInput{[]byte{}, -1, []byte("genesis")}
	txOut := TxOutput{100, to}
	tx := Transaction{nil, []TxInput{txIn}, []TxOutput{txOut}}
	tx.ID = tx.Hash()
	return &tx
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx Transaction) Serialize() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}