package blockchain

import (
	"log"
	"quantum-blockchain/p2p"
	"time"
)

func (bc *Blockchain) StartSyncHandler(net *p2p.Network) {
	for msg := range net.MessageChan {
		switch msg.Type {
		case p2p.MsgBlock:
			var blockMsg p2p.BlockMessage
			if err := decodeGob(msg.Payload, &blockMsg); err != nil {
				log.Printf("Block decode error: %v", err)
				continue
			}
			
			if err := bc.ProcessBlock(blockMsg.Block); err != nil {
				log.Printf("Block processing failed: %v", err)
			}
			
		case p2p.MsgTx:
			var txMsg p2p.TxMessage
			if err := decodeGob(msg.Payload, &txMsg); err != nil {
				log.Printf("Transaction decode error: %v", err)
				continue
			}
			
			bc.Mempool.AddTransaction(txMsg.Tx)
			
		case p2p.MsgGetBlock:
			var req p2p.GetBlockMessage
			if err := decodeGob(msg.Payload, &req); err != nil {
				log.Printf("GetBlock decode error: %v", err)
				continue
			}
			
			block, err := bc.GetBlockByHash(req.Hash)
			if err != nil {
				continue
			}
			
			blockMsg := p2p.BlockMessage{Block: block}
			net.Broadcast(p2p.MsgBlock, encodeGob(blockMsg))
		}
	}
}

func (bc *Blockchain) ProcessBlock(block *Block) error {
	lastBlock, err := bc.GetLastBlock()
	if err != nil {
		return err
	}
	
	// Проверка последовательности
	if !bytes.Equal(block.PrevHash, lastBlock.Hash) {
		log.Printf("Blockchain fork detected! Trying to sync...")
		return bc.handleFork(block)
	}
	
	// Добавление блока
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	return bc.Database.Update(func(txn *badger.Txn) error {
		return txn.Set(block.Hash, block.Serialize())
	})
}

func (bc *Blockchain) handleFork(newBlock *Block) error {
	// Упрощенная реализация: всегда принимаем самую длинную цепь
	// В реальном проекте нужно сравнивать сложность
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	return bc.Database.Update(func(txn *badger.Txn) error {
		// Удаляем блоки от точки расхождения
		// ... (реализация зависит от структуры данных)
		
		// Добавляем новые блоки
		return txn.Set(newBlock.Hash, newBlock.Serialize())
	})
}

// Вспомогательные функции кодирования
func encodeGob(v interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil
	}
	return buf.Bytes()
}

func decodeGob(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}